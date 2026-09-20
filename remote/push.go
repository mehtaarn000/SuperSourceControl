package remote

import (
	"context"
	"fmt"
	"ssc/core"
)

// Push publishes the active branch, using the observed remote tip as a lease.
func Push(ctx context.Context, r *Repository, c *Client) error {
	unlock, err := core.LockRepository(r.Root)
	if err != nil {
		return err
	}
	defer unlock()
	branch, err := r.Branch()
	if err != nil {
		return err
	}
	history, err := r.History(branch)
	if err != nil {
		return err
	}
	next := tip(history)
	if next == "" {
		return fmt.Errorf("make a commit before pushing")
	}
	refs, err := c.Refs(ctx)
	if err != nil {
		return err
	}
	g, err := loadGraph(ctx, []string{next}, r.readObject)
	if err != nil {
		return err
	}
	old := refs[branch]
	if old == next {
		return nil
	}
	if old != "" && !g.ancestor(old, next) {
		return fmt.Errorf("remote is ahead, divergent, or beyond a legacy history boundary; pull before pushing (merging is not supported)")
	}
	for hash, obj := range g {
		has, err := c.Has(ctx, hash, obj.Kind)
		if err != nil {
			return err
		}
		if !has {
			if err := c.Put(ctx, hash, obj.Kind, obj.Data); err != nil {
				return err
			}
		}
	}
	return c.Update(ctx, branch, old, next)
}
