package provision

import (
	"fmt"
	"regexp"
	"strings"
)

const MinPasswordLength = 12

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)

type Request struct {
	RoleName     string
	DatabaseName string
	RolePassword string
}

type FieldError struct {
	Field   string
	Message string
}

func (r Request) Normalized() Request {
	return Request{
		RoleName:     strings.TrimSpace(r.RoleName),
		DatabaseName: strings.TrimSpace(r.DatabaseName),
		RolePassword: r.RolePassword,
	}
}

func ValidateRequest(req Request) []FieldError {
	req = req.Normalized()
	var errs []FieldError

	if err := ValidateIdentifier("业务用户名", req.RoleName); err != nil {
		errs = append(errs, FieldError{Field: "role_name", Message: err.Error()})
	}
	if err := ValidateIdentifier("数据库名", req.DatabaseName); err != nil {
		errs = append(errs, FieldError{Field: "database_name", Message: err.Error()})
	}
	if isReservedDatabase(req.DatabaseName) {
		errs = append(errs, FieldError{Field: "database_name", Message: "数据库名不能使用 postgres、template0 或 template1"})
	}
	if len(req.RolePassword) < MinPasswordLength {
		errs = append(errs, FieldError{Field: "role_password", Message: fmt.Sprintf("业务用户密码至少需要 %d 字符", MinPasswordLength)})
	}
	return errs
}

func ValidateIdentifier(label, value string) error {
	if value == "" {
		return fmt.Errorf("%s不能为空", label)
	}
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("%s只能使用字母、数字和下划线，且必须以字母或下划线开头，最长 63 字符", label)
	}
	return nil
}

func isReservedDatabase(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "postgres", "template0", "template1":
		return true
	default:
		return false
	}
}
