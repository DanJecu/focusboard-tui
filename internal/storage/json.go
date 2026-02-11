package storage

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/danjecu/focusboard-tui/internal/model"
)

func Load(path string) (model.Store, error) {
	var s model.Store
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}

	for i := range s.Projects {
		if s.Projects[i].Todos == nil {
			s.Projects[i].Todos = []model.Todo{}
		}
	}
	return s, nil
}

func Save(path string, s model.Store) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
