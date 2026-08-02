# frozen_string_literal: true

require "fileutils"
require "minitest/autorun"
require "tmpdir"
require_relative "../scripts/validate_slices"

class SpecValidatorTest < Minitest::Test
  REQUIRED_MARKDOWN = {
    "README.md" => "# Test Slice\n",
    "permission-matrix.md" => "# Permission Matrix\n",
    "state-transitions.md" => "# State Transitions\n",
    "data-mapping.md" => "# Data Mapping\n",
    "acceptance-scenarios.md" => "# Acceptance Scenarios\n",
    "parity-tests.md" => "# Parity Tests\n"
  }.freeze

  def test_specification_allows_pending_gates_as_warnings
    with_slice(status: "specification") do |root, _slice|
      result, count = VisionOpus::SliceValidator.new(root).validate_all

      assert_equal 1, count
      assert_empty result.errors
      assert result.warnings.any? { |warning| warning.include?("security approval") }
      assert result.warnings.any? { |warning| warning.include?("TEST-Q-001") }
    end
  end

  def test_ready_slice_rejects_pending_approval_and_open_question
    with_slice(status: "ready") do |root, _slice|
      result, = VisionOpus::SliceValidator.new(root).validate_all

      refute result.success?
      assert result.errors.any? { |error| error.include?("security approval must be approved") }
      assert result.errors.any? { |error| error.include?("TEST-Q-001") }
      assert result.errors.any? { |error| error.include?("source group") }
      assert result.errors.any? { |error| error.include?("task") }
    end
  end

  def test_invalid_citation_is_rejected
    with_slice(status: "specification") do |root, slice|
      evidence_path = slice.join("evidence", "TEST-CLAIM-0001.yaml")
      evidence_path.write(evidence_path.read.sub("  lines: 10-20\n", "  lines:\n"))

      result, = VisionOpus::SliceValidator.new(root).validate_all

      assert result.errors.any? { |error| error.include?("source.lines is required") }
    end
  end

  def test_ready_slice_passes_with_completed_gates
    with_slice(status: "ready", complete: true) do |root, _slice|
      result, = VisionOpus::SliceValidator.new(root).validate_all

      assert_empty result.errors
      assert_empty result.warnings
    end
  end

  def test_claim_ids_outside_status_sections_are_not_treated_as_claim_rows
    with_slice(status: "specification") do |root, slice|
      path = slice.join("verified-behaviour.md")
      path.write(path.read + "\n## Verification Queue\n\nTEST-CLAIM-9999 remains queued.\n")

      result, = VisionOpus::SliceValidator.new(root).validate_all

      assert_empty result.errors
    end
  end

  private

  def with_slice(status:, complete: false)
    Dir.mktmpdir("visionopus-spec-validator") do |directory|
      root = Pathname(directory).join("doc", "slices")
      slice = root.join("01-test-slice")
      FileUtils.mkdir_p(slice.join("evidence"))
      FileUtils.mkdir_p(slice.join("verification"))
      REQUIRED_MARKDOWN.each { |name, content| slice.join(name).write(content) }
      write_fixture(slice, status: status, complete: complete)
      yield root, slice
    end
  end

  def write_fixture(slice, status:, complete:)
    approval = {
      "area" => "security",
      "status" => complete ? "approved" : "pending",
      "approver_kind" => complete ? "human" : nil,
      "approver" => complete ? "Security Owner" : nil,
      "approved_at" => complete ? "2026-08-02" : nil,
      "evidence" => complete ? "doc/decisions/0001-test.md" : nil,
      "notes" => complete ? "Approved for the test fixture." : "Pending security review."
    }
    metadata = {
      "schema_version" => 1,
      "slice" => "01-test-slice",
      "title" => "Test Slice",
      "status" => status,
      "risk" => {
        "tier" => "high",
        "domains" => ["security"],
        "reasons" => ["Tests a security boundary."]
      },
      "owners" => { "engineering" => "test-owner", "product" => "test-owner" },
      "approvals" => [approval]
    }
    slice.join("slice.yaml").write(metadata.to_yaml)

    source_status = complete ? "verified" : "needs_verification"
    slice.join("sources.yaml").write(<<~YAML)
      slice: 01-test-slice
      status: #{status}
      source_groups:
        - id: test-source
          purpose: Test source
          status: #{source_status}
          sources:
            - path: legacy/Test.php
              symbol: Test::run
              lines: 10-20
    YAML

    task_status = complete ? "completed" : "in_progress"
    slice.join("work-queue.yaml").write(<<~YAML)
      slice: 01-test-slice
      status: #{status}
      tasks:
        - id: TEST-001
          title: Test task
          status: #{task_status}
    YAML

    slice.join("evidence", "TEST-CLAIM-0001.yaml").write(<<~YAML)
      id: TEST-CLAIM-0001
      task: TEST-001
      claim: The legacy test performs the behaviour.
      authored_by: evidence-author
      authored_at: "2026-08-01"
      legacy_revision: abc123
      source:
        path: legacy/Test.php
        symbol: Test::run
        lines: 10-20
      evidence_type: source
      confidence: needs_independent_verification
      open_questions: []
    YAML

    slice.join("verification", "TEST-CLAIM-0001.yaml").write(<<~YAML)
      id: TEST-CLAIM-0001
      status: accepted
      verified_by: independent-reviewer
      verified_at: "2026-08-02"
      reviewer_kind: human
      independent_of_author: true
      legacy_revision: abc123
      source_checked:
        path: legacy/Test.php
        symbol: Test::run
        lines: 10-20
      notes: Source location checked independently.
    YAML

    slice.join("verified-behaviour.md").write(<<~MARKDOWN)
      # Verified Behaviour

      ## Accepted Claims

      | Claim ID | Behaviour | Source | Verification |
      | --- | --- | --- | --- |
      | TEST-CLAIM-0001 | Test behaviour | Test.php:10-20 | Accepted |
    MARKDOWN

    question_status = complete ? "Resolved" : "Open"
    slice.join("open-questions.md").write(<<~MARKDOWN)
      # Open Questions

      ## Blocking Questions

      | ID | Question | Owner | Next action | Status |
      | --- | --- | --- | --- | --- |
      | TEST-Q-001 | Is this resolved? | Test | Decide | #{question_status} |
    MARKDOWN

    slice.join("api-contract.yaml").write(<<~YAML)
      openapi: 3.1.0
      info:
        title: Test API
        version: 0.0.0
      paths: {}
    YAML
  end
end
