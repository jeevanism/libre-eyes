#!/usr/bin/env ruby
# frozen_string_literal: true

require "date"
require "pathname"
require "set"
require "yaml"

module VisionOpus
  class ValidationResult
    attr_reader :errors, :warnings

    def initialize
      @errors = []
      @warnings = []
    end

    def error(path, message)
      @errors << "#{path}: #{message}"
    end

    def warning(path, message)
      @warnings << "#{path}: #{message}"
    end

    def merge(other)
      @errors.concat(other.errors)
      @warnings.concat(other.warnings)
      self
    end

    def success?
      errors.empty?
    end
  end

  class SliceValidator
    REQUIRED_FILES = %w[
      slice.yaml
      README.md
      sources.yaml
      work-queue.yaml
      verified-behaviour.md
      permission-matrix.md
      state-transitions.md
      data-mapping.md
      acceptance-scenarios.md
      api-contract.yaml
      parity-tests.md
      open-questions.md
    ].freeze

    LIFECYCLE_STATUSES = %w[
      discovery specification ready implementation complete deferred
    ].freeze
    READY_STATUSES = %w[ready implementation complete].freeze
    RISK_TIERS = %w[critical high standard foundation].freeze
    DECISION_DOMAINS = %w[
      clinical prescribing consent migration security privacy legal product
      operations accessibility
    ].freeze
    MANDATORY_HUMAN_APPROVALS = %w[
      clinical prescribing consent migration security
    ].freeze
    APPROVAL_STATUSES = %w[pending approved rejected not_required].freeze
    SOURCE_GROUP_STATUSES = %w[
      discovery_required needs_verification verified excluded
    ].freeze
    VERIFICATION_STATUSES = %w[accepted rejected uncertain].freeze
    EVIDENCE_CONFIDENCE = %w[
      needs_independent_verification verified rejected uncertain
    ].freeze
    TERMINAL_TASK_STATUSES = %w[completed deferred cancelled].freeze
    RESOLVED_QUESTION_STATUSES = %w[
      closed resolved deferred not_applicable not-applicable
    ].freeze
    CLAIM_ID_PATTERN = /\b[A-Z][A-Z0-9-]*-CLAIM-\d+\b/

    attr_reader :root

    def initialize(root)
      @root = Pathname(root).expand_path
    end

    def validate_all
      result = ValidationResult.new
      unless root.directory?
        result.error(root, "slice root does not exist")
        return [result, 0]
      end

      slice_dirs = root.children.select(&:directory?).sort
      if slice_dirs.empty?
        result.error(root, "no slice directories found")
        return [result, 0]
      end

      slice_dirs.each { |slice_dir| result.merge(validate_slice(slice_dir)) }
      [result, slice_dirs.length]
    end

    def validate_slice(slice_dir)
      result = ValidationResult.new
      cache = {}

      validate_required_files(slice_dir, result)
      validate_yaml_files(slice_dir, result, cache)

      metadata = cache[slice_dir.join("slice.yaml")]
      return result unless metadata.is_a?(Hash)

      lifecycle = metadata["status"].to_s
      readiness_gate = READY_STATUSES.include?(lifecycle)

      validate_metadata(slice_dir, metadata, readiness_gate, result)
      validate_slice_identity(slice_dir, metadata, cache, result)
      validate_sources(slice_dir, cache, readiness_gate, result)
      validate_work_queue(slice_dir, cache, readiness_gate, result)
      validate_evidence(slice_dir, cache, readiness_gate, result)
      validate_blocking_questions(slice_dir, readiness_gate, result)

      result
    end

    private

    def validate_required_files(slice_dir, result)
      REQUIRED_FILES.each do |name|
        path = slice_dir.join(name)
        result.error(relative(path), "required file is missing") unless path.file?
      end

      %w[evidence verification].each do |name|
        path = slice_dir.join(name)
        result.error(relative(path), "required directory is missing") unless path.directory?
      end
    end

    def validate_yaml_files(slice_dir, result, cache)
      slice_dir.glob("**/*.yaml").sort.each do |path|
        cache[path] = load_yaml(path, result)
      end
    end

    def load_yaml(path, result)
      YAML.safe_load(
        path.read,
        permitted_classes: [Date, Time],
        permitted_symbols: [],
        aliases: false
      ) || {}
    rescue Psych::Exception => error
      result.error(relative(path), "invalid YAML: #{error.message.lines.first.strip}")
      nil
    rescue SystemCallError => error
      result.error(relative(path), "cannot read YAML: #{error.message}")
      nil
    end

    def validate_metadata(slice_dir, metadata, readiness_gate, result)
      path = slice_dir.join("slice.yaml")
      require_integer(metadata, "schema_version", path, result)
      require_string(metadata, "slice", path, result)
      require_string(metadata, "title", path, result)

      lifecycle = metadata["status"].to_s
      validate_enum(path, "status", lifecycle, LIFECYCLE_STATUSES, result)

      risk = metadata["risk"]
      unless risk.is_a?(Hash)
        result.error(relative(path), "risk must be a mapping")
        return
      end

      tier = risk["tier"].to_s
      validate_enum(path, "risk.tier", tier, RISK_TIERS, result)

      domains = string_array(risk["domains"])
      if domains.empty?
        result.error(relative(path), "risk.domains must contain at least one domain")
      end
      unknown_domains = domains - DECISION_DOMAINS
      unless unknown_domains.empty?
        result.error(relative(path), "unknown risk domains: #{unknown_domains.join(', ')}")
      end

      reasons = string_array(risk["reasons"])
      result.error(relative(path), "risk.reasons must contain at least one reason") if reasons.empty?

      owners = metadata["owners"]
      if !owners.is_a?(Hash) || blank?(owners["engineering"])
        result.error(relative(path), "owners.engineering is required")
      end

      validate_approvals(path, metadata["approvals"], domains, readiness_gate, result)
    end

    def validate_approvals(path, approvals_value, domains, readiness_gate, result)
      approvals = approvals_value.is_a?(Array) ? approvals_value : []
      result.error(relative(path), "approvals must be a sequence") unless approvals_value.is_a?(Array)

      by_area = {}
      approvals.each_with_index do |approval, index|
        unless approval.is_a?(Hash)
          result.error(relative(path), "approvals[#{index}] must be a mapping")
          next
        end

        area = approval["area"].to_s
        if area.empty?
          result.error(relative(path), "approvals[#{index}].area is required")
          next
        end
        if by_area.key?(area)
          result.error(relative(path), "approval area #{area.inspect} is duplicated")
        end
        by_area[area] = approval

        validate_enum(
          path,
          "approvals[#{index}].status",
          approval["status"].to_s,
          APPROVAL_STATUSES,
          result
        )
      end

      required_areas = domains & MANDATORY_HUMAN_APPROVALS
      required_areas.each do |area|
        approval = by_area[area]
        unless approval
          result.error(relative(path), "mandatory human approval #{area.inspect} is missing")
          next
        end

        if readiness_gate
          validate_approved_human_decision(path, area, approval, result)
        elsif approval["status"] != "approved"
          result.warning(
            relative(path),
            "#{area} approval is #{approval['status'].inspect}; it will block readiness"
          )
        end
      end
    end

    def validate_approved_human_decision(path, area, approval, result)
      unless approval["status"] == "approved"
        result.error(relative(path), "#{area} approval must be approved before readiness")
        return
      end

      unless approval["approver_kind"] == "human"
        result.error(relative(path), "#{area} approval must have approver_kind: human")
      end
      require_nested_string(approval, "approver", path, "#{area} approval", result)
      require_nested_string(approval, "evidence", path, "#{area} approval", result)
      require_nested_string(approval, "notes", path, "#{area} approval", result)

      approved_at = approval["approved_at"]
      if blank?(approved_at)
        result.error(relative(path), "#{area} approval approved_at is required")
      else
        begin
          Date.iso8601(approved_at.to_s)
        rescue Date::Error
          result.error(relative(path), "#{area} approval approved_at must be an ISO 8601 date")
        end
      end
    end

    def validate_slice_identity(slice_dir, metadata, cache, result)
      expected = slice_dir.basename.to_s
      metadata_id = metadata["slice"].to_s
      if metadata_id != expected
        result.error(relative(slice_dir.join("slice.yaml")), "slice must match directory #{expected.inspect}")
      end

      %w[sources.yaml work-queue.yaml].each do |name|
        path = slice_dir.join(name)
        document = cache[path]
        next unless document.is_a?(Hash)

        if document["slice"].to_s != metadata_id
          result.error(relative(path), "slice does not match slice.yaml")
        end
        if document["status"].to_s != metadata["status"].to_s
          result.error(relative(path), "status does not match slice.yaml")
        end
      end
    end

    def validate_sources(slice_dir, cache, readiness_gate, result)
      path = slice_dir.join("sources.yaml")
      document = cache[path]
      return unless document.is_a?(Hash)

      groups = document["source_groups"]
      unless groups.is_a?(Array) && !groups.empty?
        result.error(relative(path), "source_groups must contain at least one group")
        return
      end

      ids = Set.new
      groups.each_with_index do |group, index|
        unless group.is_a?(Hash)
          result.error(relative(path), "source_groups[#{index}] must be a mapping")
          next
        end

        id = group["id"].to_s
        if id.empty?
          result.error(relative(path), "source_groups[#{index}].id is required")
        elsif !ids.add?(id)
          result.error(relative(path), "source group #{id.inspect} is duplicated")
        end

        status = group["status"].to_s
        validate_enum(path, "source group #{id} status", status, SOURCE_GROUP_STATUSES, result)
        if readiness_gate && !%w[verified excluded].include?(status)
          result.error(relative(path), "source group #{id.inspect} is not verified or excluded")
        end

        sources = group["sources"]
        if status != "excluded" && (!sources.is_a?(Array) || sources.empty?)
          result.error(relative(path), "source group #{id.inspect} must contain sources")
        end
      end
    end

    def validate_work_queue(slice_dir, cache, readiness_gate, result)
      path = slice_dir.join("work-queue.yaml")
      document = cache[path]
      return unless document.is_a?(Hash)

      tasks = document["tasks"]
      unless tasks.is_a?(Array) && !tasks.empty?
        result.error(relative(path), "tasks must contain at least one task")
        return
      end

      ids = Set.new
      tasks.each_with_index do |task, index|
        unless task.is_a?(Hash)
          result.error(relative(path), "tasks[#{index}] must be a mapping")
          next
        end

        id = task["id"].to_s
        if id.empty?
          result.error(relative(path), "tasks[#{index}].id is required")
        elsif !ids.add?(id)
          result.error(relative(path), "task #{id.inspect} is duplicated")
        end

        if readiness_gate && !TERMINAL_TASK_STATUSES.include?(task["status"].to_s)
          result.error(relative(path), "task #{id.inspect} is not complete or explicitly deferred")
        end
      end
    end

    def validate_evidence(slice_dir, cache, readiness_gate, result)
      evidence = records_by_id(slice_dir.join("evidence"), cache, "evidence", result)
      verification = records_by_id(slice_dir.join("verification"), cache, "verification", result)

      evidence.each do |id, record|
        path = record.fetch(:path)
        data = record.fetch(:data)
        validate_evidence_record(path, data, readiness_gate, result)

        review = verification[id]
        if review.nil?
          message = "claim #{id} has no verification record"
          readiness_gate ? result.error(relative(path), message) : result.warning(relative(path), "#{message}; it will block readiness")
          next
        end

        validate_source_alignment(id, data, review.fetch(:data), path, review.fetch(:path), result)
      end

      verification.each do |id, record|
        path = record.fetch(:path)
        data = record.fetch(:data)
        validate_verification_record(path, data, readiness_gate, result)
        result.error(relative(path), "verification #{id} has no evidence record") unless evidence.key?(id)
      end

      validate_verified_behaviour(slice_dir, evidence, verification, result)
    end

    def records_by_id(directory, cache, kind, result)
      records = {}
      return records unless directory.directory?

      directory.glob("*.yaml").sort.each do |path|
        data = cache[path]
        next unless data.is_a?(Hash)

        id = data["id"].to_s
        if id.empty?
          result.error(relative(path), "#{kind} id is required")
          next
        end
        if path.basename(".yaml").to_s != id
          result.error(relative(path), "filename must match id #{id.inspect}")
        end
        if records.key?(id)
          result.error(relative(path), "duplicate #{kind} id #{id.inspect}")
        end
        records[id] = { path: path, data: data }
      end
      records
    end

    def validate_evidence_record(path, data, readiness_gate, result)
      require_string(data, "claim", path, result)
      validate_enum(path, "confidence", data["confidence"].to_s, EVIDENCE_CONFIDENCE, result)
      validate_citation(path, data["source"], "source", result)
      validate_optional_citations(path, data["supporting_sources"], "supporting_sources", result)

      return unless readiness_gate

      require_string(data, "authored_by", path, result)
      require_string(data, "authored_at", path, result)
      require_string(data, "legacy_revision", path, result)
    end

    def validate_verification_record(path, data, readiness_gate, result)
      validate_enum(path, "status", data["status"].to_s, VERIFICATION_STATUSES, result)
      require_string(data, "verified_by", path, result)
      require_string(data, "notes", path, result)
      validate_citation(path, data["source_checked"], "source_checked", result)
      validate_optional_citations(
        path,
        data["supporting_sources_checked"],
        "supporting_sources_checked",
        result
      )

      return unless readiness_gate

      require_string(data, "verified_at", path, result)
      require_string(data, "reviewer_kind", path, result)
      require_string(data, "legacy_revision", path, result)
      unless data["independent_of_author"] == true
        result.error(relative(path), "independent_of_author must be true before readiness")
      end
    end

    def validate_citation(path, citation, field, result)
      unless citation.is_a?(Hash)
        result.error(relative(path), "#{field} must be a mapping")
        return
      end

      %w[path symbol lines].each do |key|
        if blank?(citation[key])
          result.error(relative(path), "#{field}.#{key} is required")
        end
      end
    end

    def validate_optional_citations(path, citations, field, result)
      return if citations.nil?
      unless citations.is_a?(Array)
        result.error(relative(path), "#{field} must be a sequence")
        return
      end

      citations.each_with_index do |citation, index|
        validate_citation(path, citation, "#{field}[#{index}]", result)
      end
    end

    def validate_source_alignment(id, evidence, verification, evidence_path, verification_path, result)
      source = evidence["source"]
      checked = verification["source_checked"]
      return unless source.is_a?(Hash) && checked.is_a?(Hash)

      %w[path symbol lines].each do |field|
        next if source[field].to_s == checked[field].to_s

        result.error(
          relative(verification_path),
          "#{id} source_checked.#{field} does not match #{relative(evidence_path)}"
        )
      end
    end

    def validate_verified_behaviour(slice_dir, evidence, verification, result)
      path = slice_dir.join("verified-behaviour.md")
      return unless path.file?

      text = path.read
      listed_by_status = {
        "accepted" => claim_ids_in_section(text, "Accepted Claims"),
        "rejected" => claim_ids_in_section(text, "Rejected Claims"),
        "uncertain" => claim_ids_in_section(text, "Uncertain Claims")
      }

      listed_by_status.each do |expected_status, claim_ids|
        claim_ids.each do |id|
          unless evidence.key?(id)
            result.error(relative(path), "#{expected_status} claim #{id} has no evidence record")
            next
          end
          review = verification[id]
          if review.nil?
            result.error(relative(path), "#{expected_status} claim #{id} has no verification record")
          elsif review.fetch(:data)["status"] != expected_status
            result.error(
              relative(path),
              "#{expected_status} claim #{id} verification has status #{review.fetch(:data)['status'].inspect}"
            )
          end
        end
      end

      verification.each do |id, record|
        status = record.fetch(:data)["status"]
        next unless VERIFICATION_STATUSES.include?(status)
        next if listed_by_status.fetch(status).include?(id)

        result.error(relative(path), "#{status} verification #{id} is missing from the #{status} claims section")
      end
    rescue SystemCallError => error
      result.error(relative(path), "cannot read verified behaviour: #{error.message}")
    end

    def claim_ids_in_section(text, heading)
      lines = text.lines
      start = lines.index { |line| line.strip.casecmp("## #{heading}").zero? }
      return [] unless start

      lines[(start + 1)..]
        .take_while { |line| !line.start_with?("## ") }
        .join
        .scan(CLAIM_ID_PATTERN)
        .uniq
    end

    def validate_blocking_questions(slice_dir, readiness_gate, result)
      path = slice_dir.join("open-questions.md")
      return unless path.file?

      lines = path.readlines(chomp: true)
      start = lines.index { |line| line.strip.casecmp("## Blocking Questions").zero? }
      return result.error(relative(path), "Blocking Questions section is missing") unless start

      section = lines[(start + 1)..].take_while { |line| !line.start_with?("## ") }
      open_rows = section.filter_map do |line|
        next unless line.start_with?("|")

        cells = line.split("|").map(&:strip).reject(&:empty?)
        next if cells.empty? || cells.first.casecmp("ID").zero? || cells.first.match?(/^-+$/)
        next if cells.length < 2

        status = cells.last.downcase.tr(" ", "_")
        cells.first unless RESOLVED_QUESTION_STATUSES.include?(status)
      end

      return if open_rows.empty?

      message = "unresolved blocking questions: #{open_rows.join(', ')}"
      readiness_gate ? result.error(relative(path), message) : result.warning(relative(path), "#{message}; they will block readiness")
    rescue SystemCallError => error
      result.error(relative(path), "cannot read open questions: #{error.message}")
    end

    def require_integer(document, key, path, result)
      return if document[key].is_a?(Integer)

      result.error(relative(path), "#{key} must be an integer")
    end

    def require_string(document, key, path, result)
      return unless blank?(document[key])

      result.error(relative(path), "#{key} is required")
    end

    def require_nested_string(document, key, path, prefix, result)
      return unless blank?(document[key])

      result.error(relative(path), "#{prefix} #{key} is required")
    end

    def validate_enum(path, field, value, allowed, result)
      return if allowed.include?(value)

      result.error(relative(path), "#{field} must be one of: #{allowed.join(', ')}")
    end

    def string_array(value)
      return [] unless value.is_a?(Array)

      value.map(&:to_s).reject(&:empty?)
    end

    def blank?(value)
      value.nil? || value.to_s.strip.empty?
    end

    def relative(path)
      path.relative_path_from(root.parent.parent).to_s
    rescue ArgumentError
      path.to_s
    end
  end

  def self.run(argv, stdout: $stdout, stderr: $stderr)
    default_root = Pathname(__dir__).join("..", "doc", "slices").expand_path
    root = argv.empty? ? default_root : Pathname(argv.fetch(0))
    validator = SliceValidator.new(root)
    result, count = validator.validate_all

    result.warnings.each { |warning| stderr.puts("WARNING: #{warning}") }
    result.errors.each { |error| stderr.puts("ERROR: #{error}") }
    stdout.puts("Validated #{count} slice(s): #{result.errors.length} error(s), #{result.warnings.length} warning(s).")

    result.success? ? 0 : 1
  end
end

exit VisionOpus.run(ARGV) if $PROGRAM_NAME == __FILE__
