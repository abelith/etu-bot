package handlers

import "regexp"

var nameRe = regexp.MustCompile(`^[А-ЯЁа-яёA-Za-z0-9]+(?:[ -][А-ЯЁа-яёA-Za-z0-9]+)*$`)

func ValidateName(name string) (bool, error) {
	return nameRe.MatchString(name), nil
}
