package controllers


// Request types
type CreateEventRequest struct {
	// Name of the event
	Name string `json:"name" validate:"required" example:"Tech Conference 2023"`
	// Description of the event
	Description string `json:"description" example:"Annual tech conference with various speakers and sessions"`
	// Date of the event
	Date string `json:"date" validate:"required" example:"2023-10-01T10:00:00Z"`
	// Location of the event
	Location string `json:"location" validate:"required" example:"San Francisco, CA"`
}


type CreateEventResponse struct {
	// ID of the created event
	ID string `json:"id" example:"12345"`
	// Name of the event
	Name string `json:"name" example:"Tech Conference 2023"`
	// Description of the event
	Description string `json:"description" example:"Annual tech conference with various speakers and sessions"`
}