package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type CommentChange struct {
	ChangeType ChangeType
	Comment    parser.Comment
	OldComment *parser.Comment
}

func (c *CommentChange) SQL() string {
	objectRef := c.getObjectRef(c.Comment)

	switch c.ChangeType {
	case CreateComment:
		return fmt.Sprintf("COMMENT ON %s IS %s;", objectRef, quoteLiteral(c.Comment.Comment))
	case DropComment:
		return fmt.Sprintf("COMMENT ON %s IS NULL;", objectRef)
	}
	return ""
}

func (c *CommentChange) DownSQL() string {
	switch c.ChangeType {
	case CreateComment:
		objectRef := c.getObjectRef(c.Comment)
		if c.OldComment != nil {
			return fmt.Sprintf("COMMENT ON %s IS %s;", objectRef, quoteLiteral(c.OldComment.Comment))
		}
		return fmt.Sprintf("COMMENT ON %s IS NULL;", objectRef)
	case DropComment:
		if c.OldComment != nil {
			objectRef := c.getObjectRef(*c.OldComment)
			return fmt.Sprintf("COMMENT ON %s IS %s;", objectRef, quoteLiteral(c.OldComment.Comment))
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped comment on %s", c.Comment.Name)
	}
	return ""
}

func (c *CommentChange) Type() ChangeType   { return c.ChangeType }
func (c *CommentChange) ObjectName() string { return c.Comment.Name }
func (c *CommentChange) IsReversible() bool { return true }

func (c *CommentChange) getObjectRef(comment parser.Comment) string {
	switch comment.ObjectType {
	case "TABLE", "VIEW", "MATERIALIZED VIEW", "INDEX", "SEQUENCE", "FOREIGN TABLE":
		return fmt.Sprintf("%s %s", comment.ObjectType, qualifiedName(comment.Schema, comment.Name))
	case "COLUMN":
		return fmt.Sprintf("COLUMN %s.%s", qualifiedName(comment.Schema, comment.Name), quoteIdent(comment.Column))
	case "FUNCTION":
		return fmt.Sprintf("FUNCTION %s", qualifiedName(comment.Schema, comment.Name))
	case "TYPE":
		return fmt.Sprintf("TYPE %s", qualifiedName(comment.Schema, comment.Name))
	case "SCHEMA":
		return fmt.Sprintf("SCHEMA %s", quoteIdent(comment.Name))
	default:
		return fmt.Sprintf("%s %s", comment.ObjectType, qualifiedName(comment.Schema, comment.Name))
	}
}

func compareComments(current, desired []parser.Comment) []Change {
	var changes []Change

	commentKey := func(c parser.Comment) string {
		if c.Column != "" {
			return fmt.Sprintf("%s:%s:%s:%s", c.ObjectType, c.Schema, c.Name, c.Column)
		}
		return fmt.Sprintf("%s:%s:%s", c.ObjectType, c.Schema, c.Name)
	}

	currentMap := make(map[string]parser.Comment)
	for _, c := range current {
		currentMap[commentKey(c)] = c
	}
	for _, c := range desired {
		key := commentKey(c)
		if existing, exists := currentMap[key]; !exists {
			changes = append(changes, &CommentChange{ChangeType: CreateComment, Comment: c})
		} else if existing.Comment != c.Comment {
			oldCmt := existing
			changes = append(changes, &CommentChange{ChangeType: CreateComment, Comment: c, OldComment: &oldCmt})
		}
	}
	desiredMap := make(map[string]bool)
	for _, c := range desired {
		desiredMap[commentKey(c)] = true
	}
	for _, c := range current {
		if !desiredMap[commentKey(c)] {
			oldCmt := c
			changes = append(changes, &CommentChange{ChangeType: DropComment, Comment: c, OldComment: &oldCmt})
		}
	}
	return changes
}
