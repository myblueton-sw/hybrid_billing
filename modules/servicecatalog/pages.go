package servicecatalog

import "sort"

func page(ids []string, after string, limit int) ([]string, string) {
	start := sort.Search(len(ids), func(i int) bool { return ids[i] > after })
	end := start + min(limit, len(ids)-start)
	next := ""
	if end < len(ids) {
		next = ids[end-1]
	}
	return ids[start:end], next
}

func validPage(workspace, after string, limit int) bool {
	return identifier(workspace) && (after == "" || identifier(after)) && limit >= 1 && limit <= PageLimit
}

func (c *Catalog) AudiencePage(workspace, after string, limit int) ([]Audience, string, error) {
	if !c.ready() {
		return nil, "", ErrUnavailable
	}
	if !validPage(workspace, after, limit) {
		return nil, "", ErrInvalid
	}
	ids, next := page(c.ids[workspace], after, limit)
	rows := make([]Audience, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, c.audiences[audienceKey{workspace, id}].row)
	}
	return rows, next, nil
}

func (c *Catalog) ScopePage(workspace, audience, after string, limit int) ([]Scope, string, error) {
	if !c.ready() {
		return nil, "", ErrUnavailable
	}
	if !validPage(workspace, after, limit) {
		return nil, "", ErrInvalid
	}
	if _, err := c.Audience(workspace, audience); err != nil {
		return nil, "", err
	}
	found := c.audiences[audienceKey{workspace, audience}]
	ids, next := page(found.ids, after, limit)
	rows := make([]Scope, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, copyScope(found.scopes[id]))
	}
	return rows, next, nil
}
