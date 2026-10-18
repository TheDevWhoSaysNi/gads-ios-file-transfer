package transfer

// Status is written by the phone app after it handles the files in Documents.
type Status struct {
	State     string   `json:"state"`
	Processed []Result `json:"processed"`
	Errors    []Result `json:"errors"`
}

// Result is one file the app looked at during this launch.
type Result struct {
	Name        string `json:"name"`
	Destination string `json:"destination,omitempty"`
	Message     string `json:"message,omitempty"`
}

// Outcome is what we tell the person who uploaded.
type Outcome struct {
	Name    string `json:"name"`
	Where   string `json:"where"`
	Message string `json:"message,omitempty"`
}

// Describe turns a finished status into a sentence-sized outcome for one file.
func Describe(status Status, name string) Outcome {
	for _, item := range status.Errors {
		if item.Name == name {
			where := item.Destination
			if where == "" {
				where = "files"
			}
			return Outcome{Name: name, Where: where, Message: item.Message}
		}
	}
	for _, item := range status.Processed {
		if item.Name == name {
			return Outcome{Name: name, Where: item.Destination}
		}
	}
	return Outcome{Name: name, Where: "files", Message: "The file is in Files. The phone did not confirm a Photos copy."}
}
