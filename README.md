# rubric

[![CI](https://github.com/rhyselsmore/rubric/actions/workflows/ci.yml/badge.svg)](https://github.com/rhyselsmore/rubric/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rhyselsmore/rubric.svg)](https://pkg.go.dev/github.com/rhyselsmore/rubric)
[![codecov](https://codecov.io/gh/rhyselsmore/rubric/branch/main/graph/badge.svg)](https://codecov.io/gh/rhyselsmore/rubric)

A weighted scoring framework for multi-phase signal evaluation and candidate classification.

## Install

```
go get github.com/rhyselsmore/rubric
```

## Overview

Rubric lets you define scoring models as a hierarchy of **Model → Phase → Signal → Outcome**, evaluate observed signals against them, and classify candidates by comparing their scores.

| Concept            | Description                                                                          |
|--------------------|--------------------------------------------------------------------------------------|
| **Model**          | Top-level scoring rubric containing one or more phases.                              |
| **Phase**          | A weighted group of signals. The phase weight scales all signal scores within it.    |
| **Signal**         | An observable indicator with a default weight and one or more named outcomes.         |
| **Outcome**        | A possible result for a signal, each with its own weight.                            |
| **Evaluation**     | Records observed outcomes for a model's signals. Safe for concurrent use.            |
| **Score**          | Immutable result with raw and normalised (0–1) scores plus per-signal detail.        |
| **Candidates**     | Holds evaluations for multiple candidates keyed by a comparable label type.          |
| **Classification** | Result of classifying candidates — identifies the winner by highest normalised score.|

## Usage

The examples below show how to build a model that scores online content for alt-right pipeline characteristics — rhetorical techniques like us-vs-them language, narrative framing patterns like source delegitimization, and gateway potential (where subtle content that appears moderate but funnels toward extremism scores higher than overtly extreme content, because that's how the pipeline actually works).

### Define a model

```go
md := rubric.MustNewModel("pipeline", "Radicalization Pipeline Scoring",
    rubric.BuildPhase("rhetoric", "Rhetorical Techniques", 2.0,
        rubric.BuildSignal("othering", "Us-vs-Them Language", 0,
            rubric.BuildOutcome("explicit", "Overt dehumanization or scapegoating", 30),
            rubric.BuildOutcome("coded", "Dog-whistles and coded language", 15),
            rubric.BuildOutcome("absent", "No othering language detected", -10),
        ),
        rubric.BuildSignal("victimhood", "Victimhood Narrative", 0,
            rubric.BuildOutcome("central", "Persecution is the central narrative", 25),
            rubric.BuildOutcome("present", "Some victimhood framing", 10),
            rubric.BuildOutcome("absent", "No victimhood narrative", -5),
        ),
    ),
    rubric.BuildPhase("framing", "Narrative Framing", 1.5,
        rubric.BuildSignal("delegitimization", "Source Delegitimization", 0,
            rubric.BuildOutcome("blanket", "Blanket rejection of mainstream sources", 30),
            rubric.BuildOutcome("selective", "Selective distrust of specific outlets", 15),
            rubric.BuildOutcome("credible", "Cites credible, verifiable sources", -10),
        ),
        rubric.BuildSignal("gateway", "Gateway Potential", 0,
            rubric.BuildOutcome("overt", "Overtly extreme, easily identified", 15),
            rubric.BuildOutcome("subtle", "Appears moderate but funnels toward extremism", 25),
            rubric.BuildOutcome("none", "No gateway characteristics", -10),
        ),
    ),
)
```

### Evaluate and score

```go
ev := md.Evaluate()
ev.Set("rhetoric", "othering", "coded")
ev.Set("rhetoric", "victimhood", "central")
ev.Set("framing", "delegitimization", "selective")
ev.Set("framing", "gateway", "subtle")

score := ev.Score()
fmt.Printf("Normalised: %.4f\n", score.Normalized())
fmt.Printf("Raw:        %.2f\n", score.Raw())
fmt.Println(score)
```

Signals that are not explicitly set use their default weight — so you only need to record signals you've actually observed.

### Rank and triage candidates

When you have a batch of flagged content — reported posts, URLs from a crawl,
items in a moderation queue — use `Classify` to rank them by severity so
reviewers triage the worst ones first:

```go
// Three items flagged for review.
flagged := rubric.NewCandidates[string]()

// Forum post: overt othering, strong victimhood, blanket source rejection,
// but overtly extreme (easily identified, lower gateway risk).
forum := md.Evaluate()
forum.Set("rhetoric", "othering", "explicit")
forum.Set("rhetoric", "victimhood", "central")
forum.Set("framing", "delegitimization", "blanket")
forum.Set("framing", "gateway", "overt")
flagged.Add("forum-post-8821", forum)

// YouTube comment: coded language, some victimhood, selective distrust,
// and subtle gateway framing — harder to catch, higher pipeline risk.
comment := md.Evaluate()
comment.Set("rhetoric", "othering", "coded")
comment.Set("rhetoric", "victimhood", "present")
comment.Set("framing", "delegitimization", "selective")
comment.Set("framing", "gateway", "subtle")
flagged.Add("yt-comment-3304", comment)

// News article: no pipeline signals at all.
article := md.Evaluate()
article.Set("rhetoric", "othering", "absent")
article.Set("rhetoric", "victimhood", "absent")
article.Set("framing", "delegitimization", "credible")
article.Set("framing", "gateway", "none")
flagged.Add("article-1157", article)

result, err := rubric.Classify(flagged)
if err != nil {
    log.Fatal(err)
}

// Review the highest-scoring item first.
worst, score := result.Best()
fmt.Printf("Review first: %s (%.0f%% pipeline match)\n", worst, score.Normalized()*100)
```

`Candidates` is generic — any `comparable` type works as a key (database IDs,
URLs, enum values, etc.):

```go
candidates := rubric.NewCandidates[int]()
candidates.Add(8821, ev1)
candidates.Add(3304, ev2)
```

## How scoring works

### Raw score

For each phase, the weights of all signal outcomes are summed (unset signals
use their default weight), then multiplied by the phase weight. The raw score
is the sum across all phases:

```
raw = Σ phase.weight × Σ outcome_weight(signal)
```

Phase weights are **scaling factors**, not proportions that must sum to 1. A
phase with weight 2.0 literally doubles its signals' contribution compared to
a phase with weight 1.0.

### Min and Max bounds

At construction time each signal computes its minimum and maximum possible
weight by scanning all outcome weights (including the default). The theoretical
bounds of the entire model are:

```
min = Σ phase.weight × Σ signal.minWeight
max = Σ phase.weight × Σ signal.maxWeight
```

### Normalisation

The normalised score maps the raw score to [0, 1] using min-max normalisation:

```
normalised = (raw - min) / (max - min)
```

- A normalised score of **0** means every signal is at its lowest possible weight.
- A normalised score of **1** means every signal is at its highest possible weight.

### Edge Cases

| Scenario | Behaviour |
|---|---|
| Signal not set | Uses the signal's default weight (the `defaultWeight` passed to `NewSignal`). |
| All signals unset | Still produces a valid score — every signal falls back to its default. |
| `min == max` (all outcomes have the same weight) | Normalised returns **1.0**. There is no score range, so you are always at the maximum. |
| Negative outcome weights | Fully supported. A signal can have negative weights to penalise certain outcomes. |
| Overwriting an outcome | Calling `Set` again for the same phase/signal replaces the previous outcome. |

## Concurrency

`Evaluation` is safe for concurrent use — `Set` and `Score` can be called from multiple goroutines. `Candidates.Add` is also safe for concurrent use.

## License

[MIT](LICENSE)
