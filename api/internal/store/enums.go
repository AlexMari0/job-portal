package store

// Allowed values, the enums of the Mongoose schemas (and of api/openapi.yaml).
var (
	Locations  = []string{"Bangalore", "Washington", "Hyderabad", "Mumbai", "California", "Chennai", "New York"}
	Categories = []string{"Programming", "Data Science", "Designing", "Networking", "Management", "Marketing", "Cybersecurity"}
	Levels     = []string{"Beginner level", "Intermediate level", "Senior level"}
	Statuses   = []string{StatusPending, StatusAccepted, StatusRejected}
)
