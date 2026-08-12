import { describe, expect, it } from 'vitest'

import { eyeDrawPayloadObjects } from './eyedrawRuntime'

describe('eyeDrawPayloadObjects', () => {
  it('keeps only the approved runtime doodle objects and removes legacy tags', () => {
    expect(eyeDrawPayloadObjects(`[, {"subclass":"AntSeg"}, {"tags":[]}, {"subclass":"Fundus"}, {"className":"SidePort"}]`)).toEqual([
      { className: 'AntSeg', subclass: 'AntSeg' },
      { className: 'SidePort', subclass: 'SidePort' },
    ])
  })

  it('rejects a non-array runtime serialization', () => {
    expect(() => eyeDrawPayloadObjects('{"className":"AntSeg"}')).toThrow('drawing array')
  })
})
