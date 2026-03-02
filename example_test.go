package rubric_test

import (
	"fmt"
	"log"

	"github.com/rhyselsmore/rubric"
)

// pipelineModel builds a model for scoring online content against alt-right
// pipeline characteristics. Used by multiple examples below.
func pipelineModel() *rubric.Model {
	return rubric.MustNewModel("pipeline", "Radicalization Pipeline Scoring",
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
}

func Example() {
	md := pipelineModel()

	// Score a piece of content against the pipeline model.
	ev := md.Evaluate()
	ev.Set("rhetoric", "othering", "coded")
	ev.Set("rhetoric", "victimhood", "central")
	ev.Set("framing", "delegitimization", "selective")
	ev.Set("framing", "gateway", "subtle")

	score := ev.Score()
	fmt.Printf("normalized=%.4f raw=%.1f\n", score.Normalized(), score.Raw())
	// Output:
	// normalized=0.7921 raw=140.0
}

func ExampleNewModel() {
	md, err := rubric.NewModel("pipeline", "Radicalization Pipeline Scoring",
		rubric.BuildPhase("rhetoric", "Rhetorical Techniques", 1.0,
			rubric.BuildSignal("othering", "Us-vs-Them Language", 0,
				rubric.BuildOutcome("explicit", "Overt dehumanization", 30),
				rubric.BuildOutcome("absent", "No othering detected", -10),
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(md.ID())
	// Output:
	// pipeline
}

func ExampleEvaluation_Set() {
	md := rubric.MustNewModel("pipeline", "Pipeline Scoring",
		rubric.BuildPhase("rhetoric", "Rhetorical Techniques", 1.0,
			rubric.BuildSignal("othering", "Us-vs-Them Language", 0,
				rubric.BuildOutcome("explicit", "Overt dehumanization", 30),
				rubric.BuildOutcome("absent", "No othering detected", -10),
			),
		),
	)

	ev := md.Evaluate()

	// Record an observed outcome.
	if err := ev.Set("rhetoric", "othering", "explicit"); err != nil {
		log.Fatal(err)
	}

	// Unknown signals return an error.
	err := ev.Set("rhetoric", "unknown", "explicit")
	fmt.Println(err)
	// Output:
	// rubric: signal with id "unknown" not found (phase="rhetoric")
}

func ExampleClassify() {
	md := pipelineModel()

	// A blog post with strong pipeline signals.
	post := md.Evaluate()
	post.Set("rhetoric", "othering", "explicit")
	post.Set("rhetoric", "victimhood", "central")
	post.Set("framing", "delegitimization", "blanket")
	post.Set("framing", "gateway", "subtle")

	// A news article with no pipeline signals.
	article := md.Evaluate()
	article.Set("rhetoric", "othering", "absent")
	article.Set("rhetoric", "victimhood", "absent")
	article.Set("framing", "delegitimization", "credible")
	article.Set("framing", "gateway", "none")

	candidates := rubric.NewCandidates[string]()
	candidates.Add("blog-post", post)
	candidates.Add("news-article", article)

	cl, err := rubric.Classify(candidates)
	if err != nil {
		log.Fatal(err)
	}

	winner, best := cl.Best()
	fmt.Printf("highest pipeline score: %s (%.4f)\n", winner, best.Normalized())
	// Output:
	// highest pipeline score: blog-post (1.0000)
}

func ExampleScore_Details() {
	md := rubric.MustNewModel("pipeline", "Pipeline Scoring",
		rubric.BuildPhase("rhetoric", "Rhetorical Techniques", 1.0,
			rubric.BuildSignal("othering", "Us-vs-Them Language", 0,
				rubric.BuildOutcome("explicit", "Overt dehumanization", 30),
			),
			rubric.BuildSignal("victimhood", "Victimhood Narrative", 0,
				rubric.BuildOutcome("central", "Central narrative", 25),
			),
		),
	)

	ev := md.Evaluate()
	ev.Set("rhetoric", "othering", "explicit")
	// victimhood left unset — uses default weight of 0.

	score := ev.Score()
	for _, d := range score.Details() {
		fmt.Printf("signal=%-12s outcome=%-10s weight=%3.0f default=%v\n",
			d.Signal().ID(), d.Outcome().ID(), d.Weight(), d.Outcome().IsDefault())
	}
	// Output:
	// signal=othering     outcome=explicit   weight= 30 default=false
	// signal=victimhood   outcome=default    weight=  0 default=true
}
