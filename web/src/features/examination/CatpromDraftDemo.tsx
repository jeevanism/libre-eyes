import { ClipboardList, Save } from "lucide-react";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { episodesAPI, type CatpromDemoDraftPayload } from "../../api/client";
import { describeDraftSaveError } from "./draftError";

const questions = [
  "How difficult is it to read small print?",
  "How difficult is it to recognise faces?",
  "How difficult is it to see in dim light?",
  "How difficult is it to use a computer?",
  "How difficult is it to travel independently?",
  "How satisfied are you with your vision?",
];
const answerLabels = [
  "Demo answer 1",
  "Demo answer 2",
  "Demo answer 3",
  "Demo answer 4",
];

export function CatpromDraftDemo({
  csrfToken,
  episodeId,
}: {
  csrfToken: string;
  episodeId: string;
}) {
  const [laterality, setLaterality] =
    useState<CatpromDemoDraftPayload["laterality"]>("not_applicable");
  const [answers, setAnswers] = useState(() =>
    questions.map((_, index) => ({
      questionCode: `demo_q${index + 1}`,
      answerCode: `demo_q${index + 1}_a1`,
    })),
  );
  const [comment, setComment] = useState("");
  const save = useMutation({
    mutationFn: (payload: CatpromDemoDraftPayload) =>
      episodesAPI.createCatpromDemoDraft(episodeId, csrfToken, payload),
  });
  function submit() {
    save.reset();
    save.mutate({
      recordMode: "demo_catprom",
      profileCode: "demo_catprom_v1",
      laterality,
      answers,
      comment: comment.trim(),
    });
  }
  const failure = save.isError
    ? describeDraftSaveError(save.error, "demo cataract questionnaire draft")
    : "";
  return (
    <section
      className="examination-draft-demo"
      aria-labelledby={`catprom-demo-${episodeId}`}
    >
      <div className="examination-draft-demo-heading">
        <div>
          <p>Demo</p>
          <h3 id={`catprom-demo-${episodeId}`}>Cataract questionnaire draft</h3>
        </div>
        <ClipboardList size={18} aria-hidden="true" />
      </div>
      <p className="examination-draft-demo-note">
        Synthetic questionnaire only. It is unscored and does not produce a PROM
        result, interpretation, report, or clinical record.
      </p>
      <form
        className="examination-draft-demo-form"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <label>
          <span>Demo laterality</span>
          <select
            value={laterality}
            onChange={(e) =>
              setLaterality(
                e.target.value as CatpromDemoDraftPayload["laterality"],
              )
            }
          >
            <option value="not_applicable">Not applicable</option>
            <option value="right">Right eye</option>
            <option value="left">Left eye</option>
            <option value="bilateral">Both eyes</option>
          </select>
        </label>
        <fieldset>
          <legend>Demo questions</legend>
          {questions.map((question, index) => (
            <label key={question}>
              <span>
                {index + 1}. {question}
              </span>
              <select
                aria-label={`Question ${index + 1} answer`}
                value={answers[index]?.answerCode ?? ''}
                onChange={(e) =>
                  setAnswers((current) =>
                    current.map((answer, i) =>
                      i === index
                        ? { ...answer, answerCode: e.target.value }
                        : answer,
                    ),
                  )
                }
              >
                {answerLabels.map((_, answerIndex) => (
                  <option
                    key={answerIndex}
                    value={`demo_q${index + 1}_a${answerIndex + 1}`}
                  >
                    {answerLabels[answerIndex]}
                  </option>
                ))}
              </select>
            </label>
          ))}
        </fieldset>
        <label>
          <span>Demo comment (optional)</span>
          <textarea
            maxLength={500}
            rows={3}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </label>
        {failure && (
          <p className="inline-error" role="alert">
            {failure}
          </p>
        )}
        {save.isSuccess && (
          <p className="inline-success" role="status">
            Demo cataract questionnaire draft saved. It remains uncommitted.
          </p>
        )}
        <button
          className="primary-button"
          type="submit"
          disabled={save.isPending}
        >
          <Save size={16} aria-hidden="true" />
          {save.isPending
            ? "Saving draft…"
            : "Save demo cataract questionnaire"}
        </button>
      </form>
    </section>
  );
}
