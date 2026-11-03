package sync

import "slices"

// storedRow is a local chapter or member as the matcher sees it.
type storedRow struct {
	ID      string
	BNIVMID *string
	// Keys are the normalized values a BNI VM row can match this row by.
	Keys []string
}

// remoteRow is a chapter or member from the BNI VM snapshot.
type remoteRow struct {
	ID    string
	Key   string
	Label string
}

type matches struct {
	// Local maps a BNI VM id to its stored row. An id missing here gets a new row.
	Local map[string]string
	// Linked counts rows joined by key during this run.
	Linked int
	// Ambiguous holds the labels of remote rows whose key was not unique.
	Ambiguous []string
}

// match finds the stored row behind each BNI VM row: first by a link made
// earlier, then by id, then by key. A key only links when it pairs one remote row
// with one stored row. Anything less certain stays unlinked, because a wrong link
// overwrites one person's contact details with another's, while an extra row can
// be cleaned up later.
func match(remotes []remoteRow, stored []storedRow) matches {
	byLink := make(map[string]string, len(stored))
	byID := make(map[string]bool, len(stored))
	for _, s := range stored {
		byID[s.ID] = true
		if s.BNIVMID != nil {
			byLink[*s.BNIVMID] = s.ID
		}
	}

	out := matches{Local: make(map[string]string, len(remotes))}
	claimed := map[string]bool{}
	seen := make(map[string]bool, len(remotes))
	var open []remoteRow
	for _, r := range remotes {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		local, ok := byLink[r.ID]
		if !ok && byID[r.ID] {
			local, ok = r.ID, true
		}
		if !ok {
			open = append(open, r)
			continue
		}
		out.Local[r.ID] = local
		claimed[local] = true
	}

	// Runs after every exact match, so a stored row an exact match already
	// claimed can never be handed to another BNI VM id.
	owners := map[string]map[string]bool{}
	for _, s := range stored {
		if s.BNIVMID != nil || claimed[s.ID] {
			continue
		}
		for _, k := range s.Keys {
			if owners[k] == nil {
				owners[k] = map[string]bool{}
			}
			owners[k][s.ID] = true
		}
	}

	wanted := map[string][]remoteRow{}
	for _, r := range open {
		switch rows := owners[r.Key]; {
		case r.Key == "" || len(rows) == 0:
		case len(rows) > 1:
			out.Ambiguous = append(out.Ambiguous, r.Label)
		default:
			for id := range rows {
				wanted[id] = append(wanted[id], r)
			}
		}
	}
	for id, rows := range wanted {
		if len(rows) == 1 {
			out.Local[rows[0].ID] = id
			out.Linked++
			continue
		}
		for _, r := range rows {
			out.Ambiguous = append(out.Ambiguous, r.Label)
		}
	}
	slices.Sort(out.Ambiguous)
	return out
}
