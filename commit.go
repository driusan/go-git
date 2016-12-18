package main

// Commit implements the command "git commit" in the repository pointed
// to by c.
func Commit(c *Client, args []string) (string, error) {
	// get the parent commit, if it exists
	var commitTreeArgs []string
	if parentCommit, err := c.GetHeadID(); err == nil {
		commitTreeArgs = []string{"-p", parentCommit}
	}

	// extract the message parameters that get passed directly
	//to commit-tree
	var messages []string
	var msgIncluded bool
	for idx, val := range args {
		switch val {
		case "-m", "-F":
			msgIncluded = true
			messages = append(messages, args[idx:idx+2]...)
		}
	}
	if !msgIncluded {
		s, err := getStatus(c, "# ")
		if err != nil {
			return "", err
		}

		c.GitDir.WriteFile("COMMIT_EDITMSG", []byte("\n\n"+s), 0660)
		c.ExecEditor(c.GitDir.File("COMMIT_EDITMSG"))
		commitTreeArgs = append(commitTreeArgs, "-F", c.GitDir.File("COMMIT_EDITMSG").String())
	}
	commitTreeArgs = append(commitTreeArgs, messages...)

	// write the current index tree and get the SHA1
	treeSha1 := WriteTree(c)
	commitTreeArgs = append(commitTreeArgs, treeSha1)

	// write the commit tree
	commitSha1, err := CommitTree(c, commitTreeArgs)
	if err != nil {
		return "", err
	}

	UpdateRef(c, []string{"-m", "commit from go-git", "HEAD", commitSha1})
	return commitSha1, nil
}
