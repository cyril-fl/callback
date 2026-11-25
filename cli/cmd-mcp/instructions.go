package cmdmcp

// Instructions tells an agent the cheap way through the applications, which its client puts in its system prompt.
// It holds the recipes once, so the tool descriptions stay one sentence. The writing recipes are left out when
// readOnly, whose tools cannot write.
func Instructions(readOnly bool) string {
	text := "The user's job applications. status: applied, responded, interview, offer, rejected, no_reply (the server sets it on applied, responded or interview left without news since the last follow-up, else since sent, as long as the settings say); source: spontaneous (default), job_post with source_detail its URL, recruiter with source_detail their e-mail, or any name the user gives (60 characters at most) with a free source_detail; " +
		"rating 0 to 5; days look like 2026-01-31, applied_before and applied_after are strict; interview_at is a local 2026-01-31T14:30.\n" +
		"Read as little as possible:\n" +
		"- Count before reading: application_stats, or list_applications with count_only.\n" +
		"- Filter on the server (status, source, q, applied_before, applied_after, untouched_since) rather than reading everything.\n" +
		"- list_applications is compact: ask fields only for the columns you need, page with limit and offset (50 by default).\n" +
		"- read_applications takes up to 50 ids, with max_chars to cut long comments.\n" +
		"- status_history with ids or since, not the whole history.\n" +
		"- Subagents do not receive these instructions: put the lines they need in their prompt."
	if !readOnly {
		text += "\n- Reuse a source name of application_stats rather than make a near double.\n" +
			"- Write in batches: one add_applications for all the new ones, one update_applications for a whole batch, its ops a group each, picked by ids or the filters.\n" +
			"- On filters, dry_run first to see what matches, then the real call; summary leaves the ids out when the counts are enough.\n" +
			"- The writes answer the ids and the failed items: no need to read back."
	}
	return text
}
