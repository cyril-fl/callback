package domain

// MaxBatch is the most applications one batch creates.
const MaxBatch = 500

var (
	ErrEmptyBatch = Invalid("the batch is empty")
	ErrLongBatch  = Invalid("at most 500 items per batch")
)

// BatchFailure is an item of a batch that was not written, and why. Index is its place in the items of a creation;
// Op the place of its op in an update; ID the application an update or a delete could not write.
type BatchFailure struct {
	Index *int   `json:"index,omitempty" binding:"optional"`
	Op    *int   `json:"op,omitempty" binding:"optional"`
	ID    string `json:"id,omitempty" binding:"optional"`
	Error string `json:"error"`
}

// BatchResult is what a batch wrote: the ids (left out with summary), their count, and the items that failed.
// Valid items are written even when others fail. DryRun tells that nothing was written: the ids and count are
// what the call would write.
type BatchResult struct {
	IDs    []string       `json:"ids,omitempty" binding:"optional"`
	Count  int            `json:"count"`
	Failed []BatchFailure `json:"failed,omitempty" binding:"optional"`
	DryRun bool           `json:"dry_run,omitempty" binding:"optional"`
}

// Summarize drops the ids when only the counts are asked.
func (r *BatchResult) Summarize(summary bool) {
	if summary {
		r.IDs = nil
	}
}
