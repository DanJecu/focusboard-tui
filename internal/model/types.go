package model

type Todo struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Link      string `json:"link,omitempty"`
}

type Project struct {
	Name  string `json:"name"`
	Todos []Todo `json:"todos"`
}

type Store struct {
	Projects []Project `json:"projects"`
}
