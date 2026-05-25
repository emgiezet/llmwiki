package tracker

// StalenessResult describes the freshness of one wiki entry's tracked area.
type StalenessResult struct {
	AreaName    string
	Files       []string
	StoredHash  string
	CurrentHash string
	IsStale     bool
}

// CheckFreshness re-computes the current hash for area and compares it to
// storedHash. An empty storedHash is always considered stale.
func CheckFreshness(runner GitRunner, projectRoot string, area Area, storedHash string) (StalenessResult, error) {
	currentHash, err := ComputeHash(runner, projectRoot, area.Files)
	if err != nil {
		return StalenessResult{}, err
	}

	isStale := storedHash == "" || currentHash != storedHash

	return StalenessResult{
		AreaName:    area.Name,
		Files:       area.Files,
		StoredHash:  storedHash,
		CurrentHash: currentHash,
		IsStale:     isStale,
	}, nil
}
