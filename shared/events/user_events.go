package events

const (
	UserCreatedTopic = "user.created"
	UserUpdatedTopic = "user.updated"

	UserDeletedTopic = "user.deleted"
)

type UserCreatedEvent struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UserUpdatedEvent struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type UserDeletedEvent struct {
	ID string `json:"id"`
}
