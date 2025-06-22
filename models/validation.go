package models

import "gorm.io/gorm"

// Validatable is implemented by models that can validate themselves.
type Validatable interface {
	Validate() error
}

// beforeSave is a helper that runs Validate on the given model.
func beforeSave(m any, tx *gorm.DB) error {
	if v, ok := m.(Validatable); ok {
		return v.Validate()
	}
	return nil
}
