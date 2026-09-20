package remote

import (
	"context"
	"fmt"
	"ssc/core"
	"strings"
)

type object struct {
	Kind string
	Data []byte
	Refs []core.ObjectReference
}
type graph map[string]object

func loadGraph(ctx context.Context, roots []string, load func(string, string) ([]byte, error)) (graph, error) {
	result := graph{}
	pending := make([]core.ObjectReference, 0, len(roots))
	for _, hash := range roots {
		pending = append(pending, core.ObjectReference{Hash: hash, Type: "commit"})
	}
	references, total := len(pending), 0
	if references > 10000 {
		return nil, fmt.Errorf("history exceeds transfer limits")
	}
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ref := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if existing, ok := result[ref.Hash]; ok {
			if existing.Kind != ref.Type {
				return nil, fmt.Errorf("conflicting object types")
			}
			continue
		}
		data, err := load(ref.Hash, ref.Type)
		if err != nil {
			return nil, err
		}
		id, children, err := core.InspectObject(ref.Type, data)
		if err != nil || id != ref.Hash {
			return nil, fmt.Errorf("invalid object graph")
		}
		total += len(data)
		references += len(children)
		if total > 256<<20 || references > 10000 {
			return nil, fmt.Errorf("history exceeds transfer limits")
		}
		result[ref.Hash] = object{Kind: ref.Type, Data: data, Refs: children}
		pending = append(pending, children...)
	}
	return result, nil
}
func (g graph) history(hash string) ([]string, error) {
	seen := map[string]int{}
	var result []string
	var visit func(string) error
	visit = func(id string) error {
		if seen[id] == 2 {
			return nil
		}
		if seen[id] == 1 {
			return fmt.Errorf("cycle in history")
		}
		seen[id] = 1
		obj, ok := g[id]
		if !ok || obj.Kind != "commit" {
			return fmt.Errorf("missing commit")
		}
		for _, ref := range obj.Refs {
			if ref.Type == "commit" {
				if err := visit(ref.Hash); err != nil {
					return err
				}
			}
		}
		seen[id] = 2
		result = append(result, id)
		return nil
	}
	if hash != "" {
		if err := visit(hash); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (g graph) ancestor(old, next string) bool {
	history, err := g.history(next)
	if err != nil {
		return false
	}
	for _, hash := range history {
		if hash == old {
			return true
		}
	}
	return false
}
func (g graph) files(tip string) (map[string][]byte, error) {
	result := map[string][]byte{}
	if tip == "" {
		return result, nil
	}
	c, ok := g[tip]
	if !ok || len(c.Refs) == 0 {
		return nil, fmt.Errorf("missing commit tree")
	}
	tree, ok := g[c.Refs[0].Hash]
	if !ok {
		return nil, fmt.Errorf("missing tree")
	}
	for _, line := range strings.Split(string(tree.Data), "\n") {
		if line == "" {
			continue
		}
		sep := strings.LastIndexByte(line, ' ')
		if sep < 1 {
			return nil, fmt.Errorf("invalid tree")
		}
		blob, ok := g[line[sep+1:]]
		if !ok || blob.Kind != "blob" {
			return nil, fmt.Errorf("missing blob")
		}
		result[line[:sep]] = blob.Data
	}
	return result, nil
}
