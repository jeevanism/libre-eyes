import { expect, test, type Page } from '@playwright/test'

async function openExaminationTools(page: Page) {
  const toggle = page.getByRole('button', { name: 'Examination tools' })
  if (await toggle.isVisible()) await toggle.click()
}

const session = {
  user: { id: '1', displayName: 'Synthetic Clinician' },
  context: {
    institution: { id: '1', name: 'VisionOpus Development Hospital' },
    site: { id: '1', name: 'Development Eye Clinic' },
    firm: { id: '1', name: 'Development Ophthalmology' },
  },
  permissions: ['patient.search', 'patient.summary.read', 'patient.clinical_summary.read', 'episode.read', 'event_draft.create', 'worklist.development_flow.manage'],
  csrfToken: 'synthetic-csrf-token',
  idleExpiresAt: '2026-08-06T12:00:00Z',
  absoluteExpiresAt: '2026-08-06T18:00:00Z',
  contextVersion: 1,
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/presentation/branding**', (route) => route.fulfill({ json: {
    profileVersion: 1, rowVersion: 1, status: 'published', source: 'institution',
    institutionId: 1, organizationName: 'VisionOpus Development Hospital', shortName: 'VisionOpus', browserTitle: 'VisionOpus',
    colors: { primary: '#116466', primaryHover: '#0c5355', selectedSurface: '#deefee', focus: '#0b6fcc' },
  } }))
  await page.route('**/api/v1/auth/session', (route) => route.fulfill({ json: session }))
  await page.route('**/api/v1/patients/11111111-1111-4111-8111-111111111111/summary-header', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    expect(route.request().headers()['x-context-version']).toBe('1')
    await route.fulfill({ json: {
      patientId: '11111111-1111-4111-8111-111111111111', givenName: 'Alice', familyName: 'Patient',
      dateOfBirth: '1985-04-12', ageYears: 41, gender: 'female', deceased: false,
      dateOfDeath: null, clinicalDisclosure: 'authorized', patientVersion: 1, warningVersion: 1,
    } })
  })
  await page.route('**/api/v1/patients/11111111-1111-4111-8111-111111111111/summary-header/warnings', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    await route.fulfill({ json: {
      patientId: '11111111-1111-4111-8111-111111111111',
      allergies: { status: 'present' }, alerts: { status: 'none_known' }, complete: true, warningVersion: 1,
      items: [{ kind: 'allergy', code: 'peanuts', label: 'Peanut allergy', reaction: 'Urticaria', comment: null }],
    } })
  })
  await page.route('**/api/v1/patients/11111111-1111-4111-8111-111111111111/episodes', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    await route.fulfill({ json: {
      items: [{
        id: '22222222-2222-4222-8222-222222222222', patientId: '11111111-1111-4111-8111-111111111111',
        status: 'active', startedAt: '2026-08-07T09:30:00Z', endedAt: null,
        supportServices: false, changeTracker: false, version: 1,
      }], nextCursor: null,
    } })
  })
  await page.route('**/api/v1/episodes/22222222-2222-4222-8222-222222222222/events', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    await route.fulfill({ json: {
      items: [{
        id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222',
        eventTypeCode: 'core.examination', occurredAt: '2026-08-07T10:00:00Z', status: 'current', version: 1,
      }], nextCursor: null,
    } })
  })
  await page.route('**/api/v1/episodes/22222222-2222-4222-8222-222222222222/event-drafts', async (route) => {
    const request = route.request()
    expect(request.headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    const body: unknown = request.postDataJSON()
    const eventTypeCode = (body as { eventTypeCode?: string }).eventTypeCode
    if (eventTypeCode === 'ophthalmology.visual_acuity') {
      expect(body).toEqual({
        eventTypeCode: 'ophthalmology.visual_acuity', intent: 'create', mode: 'manual', schemaVersion: 1,
        payload: {
          recordMode: 'simple',
          eyes: [
            { eye: 'right', assessment: 'recorded', readings: [{ unitCode: 'development_distance_scale', valueCode: 'development_value_m028', methodCode: 'development_unaided' }] },
            { eye: 'left', assessment: 'recorded', readings: [{ unitCode: 'development_distance_scale', valueCode: 'development_value_150', methodCode: 'development_unaided' }] },
          ],
        },
      })
    } else if (eventTypeCode === 'ophthalmology.intraocular_pressure') {
      expect(body).toEqual({
        eventTypeCode: 'ophthalmology.intraocular_pressure', intent: 'create', mode: 'manual', schemaVersion: 1,
        payload: { recordMode: 'development_raw_mmhg', profileCode: 'development_iop_manual_mmhg', eyes: [{ eye: 'right', valueCode: 'development_iop_14' }, { eye: 'left', valueCode: 'development_iop_18' }] },
      })
    } else if (eventTypeCode === 'ophthalmology.principal_diagnosis_demo') {
      expect(body).toEqual({
        eventTypeCode: 'ophthalmology.principal_diagnosis_demo', intent: 'create', mode: 'manual', schemaVersion: 1,
        payload: { recordMode: 'development_synthetic_diagnosis', profileCode: 'development_ophthalmology_diagnosis_v1', selectionCode: 'development_cataract', laterality: 'bilateral', diagnosisDate: '2026-08-09' },
      })
    } else if (eventTypeCode === 'ophthalmology.eyedraw_anterior_segment_demo') {
      const drawing = (body as { payload?: { drawing?: unknown[] } }).payload?.drawing
      expect(body).toMatchObject({
        eventTypeCode: 'ophthalmology.eyedraw_anterior_segment_demo', intent: 'create', mode: 'manual', schemaVersion: 1,
        payload: { recordMode: 'development_eyedraw_anterior_segment', canvasCode: 'development_exam_ant_seg_v1', laterality: 'right' },
      })
      expect(drawing).toEqual(expect.arrayContaining([expect.objectContaining({ className: 'AntSeg' })]))
      expect(JSON.stringify(drawing)).not.toContain('tags')
    } else {
      throw new Error(`unexpected demonstration event type: ${eventTypeCode}`)
    }
    await route.fulfill({ status: 201, json: {
      id: '77777777-7777-4777-8777-777777777777', episodeId: '22222222-2222-4222-8222-222222222222',
      eventTypeCode: 'ophthalmology.intraocular_pressure', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: { recordMode: 'development_raw_mmhg', profileCode: 'development_iop_manual_mmhg', eyes: [{ eye: 'right', valueCode: 'development_iop_14' }, { eye: 'left', valueCode: 'development_iop_18' }] }, version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false,
    } })
  })
  await page.route('**/api/v1/event-drafts/77777777-7777-4777-8777-777777777777', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    await route.fulfill({ json: {
      id: '77777777-7777-4777-8777-777777777777', episodeId: '22222222-2222-4222-8222-222222222222',
      eventTypeCode: 'ophthalmology.eyedraw_anterior_segment_demo', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: { recordMode: 'development_eyedraw_anterior_segment', canvasCode: 'development_exam_ant_seg_v1', laterality: 'right', drawing: [{ className: 'AntSeg', subclass: 'AntSeg' }] },
      version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false,
    } })
  })
  await page.route('**/api/v1/development/clinic-flow/tickets**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: {
        items: [{ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'waiting', assigneeDisplayName: null, version: 1 }],
      } })
      return
    }
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    expect(route.request().postDataJSON()).toEqual({ expectedVersion: 1 })
    await route.fulfill({ json: {
      id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'arrived', assigneeDisplayName: null, version: 2,
    } })
  })
})

test('renders the selected patient identity and warning details', async ({ page }) => {
  await page.goto('/patients/11111111-1111-4111-8111-111111111111')
  await expect(page.getByRole('heading', { name: 'Alice Patient' })).toBeVisible()
  await expect(page.getByText('12 Apr 1985')).toBeVisible()
  await expect(page.getByText('41 years')).toBeVisible()
  await expect(page.getByText('Peanut allergy')).toBeVisible()
  await expect(page.getByText('Urticaria')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Care episodes' })).toBeVisible()
  await expect(page.getByText('active')).toBeVisible()
  await page.getByRole('button', { name: 'Show events' }).click()
  await expect(page.getByText('core.examination')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open examination workspace' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Clinic flow queue' })).not.toBeVisible()
})

test('keeps clinic flow in its own operational workspace', async ({ page }) => {
  await page.goto('/clinic-flow')
  await expect(page.getByRole('heading', { name: 'Clinic flow', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Clinic flow queue' })).toBeVisible()
  await page.getByRole('button', { name: 'Mark arrived' }).click()
  await expect(page.getByText('Arrived')).toBeVisible()
})

test('keeps development examination demonstrations in a focused workspace', async ({ page }) => {
  await page.goto('/patients/11111111-1111-4111-8111-111111111111/examination')
  await expect(page.getByRole('heading', { name: 'Examination workspace' })).toBeVisible()
  await openExaminationTools(page)
  await expect(page.getByRole('button', { name: 'Visual acuity' })).toHaveAttribute('aria-current', 'page')
  const visualAcuityForm = page.locator('.visual-acuity-demo-form')
  await visualAcuityForm.getByLabel('Right eye demonstration value').selectOption('development_value_m028')
  await visualAcuityForm.getByLabel('Left eye demonstration value').selectOption('development_value_150')
  await visualAcuityForm.getByRole('button', { name: 'Save demo draft' }).click()
  await expect(visualAcuityForm.getByRole('status')).toHaveText('Demo draft saved. It remains uncommitted.')

  await page.getByRole('button', { name: 'Intraocular pressure' }).click()
  const iopForm = page.locator('.iop-draft-demo-form')
  await iopForm.getByLabel('Right eye demo IOP value').selectOption('development_iop_14')
  await iopForm.getByLabel('Left eye demo IOP value').selectOption('development_iop_18')
  await iopForm.getByRole('button', { name: 'Save demo draft' }).click()
  await expect(iopForm.getByRole('status')).toHaveText('Demo draft saved. It remains uncommitted.')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('saves only a synthetic diagnosis development draft', async ({ page }) => {
  await page.goto('/patients/11111111-1111-4111-8111-111111111111/examination?tool=selection')
  const form = page.locator('.diagnosis-draft-demo-form')
  await form.getByLabel('Demo example').selectOption('development_cataract')
  await form.getByLabel('Laterality').selectOption('bilateral')
  await form.getByLabel('Demo date').fill('2026-08-09')
  await form.getByRole('button', { name: 'Save demo draft' }).click()
  await expect(form.getByRole('status')).toHaveText('Demo draft saved. It remains uncommitted.')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('saves and recovers only the approved EyeDraw development draft', async ({ page }) => {
  await page.goto('/patients/11111111-1111-4111-8111-111111111111/examination?tool=drawing')
  const demo = page.locator('.eyedraw-draft-demo')
  await expect(demo.getByRole('heading', { name: 'Anterior segment drawing draft' })).toBeVisible()
  await expect(demo.getByRole('button', { name: 'Add anterior segment' })).toBeEnabled()
  await demo.getByRole('button', { name: 'Add anterior segment' }).click()
  await demo.getByLabel('Anterior segment pupil size').selectOption('Small')
  await demo.getByRole('button', { name: 'Save demo draft' }).click()
  await expect(demo.getByRole('status')).toHaveText('Demo drawing draft saved. It remains uncommitted.')
  await demo.getByRole('button', { name: 'Reload saved draft' }).click()
  await expect(demo.getByRole('button', { name: 'Update demo draft' })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
