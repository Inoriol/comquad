package main

func resolveComposeProjectName() string {
	if composeProjectName != "" {
		return composeProjectName
	}
	return projectName
}
