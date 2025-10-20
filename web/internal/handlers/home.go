package handlers

// HomeData is the view model for the home page.
type HomeData struct {
	Title   string
	Message string
}

// BuildHomeData constructs the default view model for the landing page.
func BuildHomeData() HomeData {
	return HomeData{
		Title:   "Hanko Field",
		Message: "Welcome to Hanko Field (Web)",
	}
}
