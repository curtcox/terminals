package scenario

import (
	"context"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func resolveRecipeTargets(ctx context.Context, env *Environment, recipe ScenarioRecipe) []string {
	if recipe.Resolve == nil {
		return nil
	}
	return recipe.Resolve(ctx, env)
}

func requestRecipeClaims(ctx context.Context, env *Environment, recipe ScenarioRecipe, targets []string) error {
	routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok || recipe.Claims == nil {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	_, err := claims.Request(ctx, recipe.Claims(targets))
	if err != nil && err != iorouter.ErrClaimConflict {
		return err
	}
	return nil
}

func applyRecipeMediaPlan(ctx context.Context, env *Environment, recipe ScenarioRecipe, targets []string) (iorouter.PlanHandle, error) {
	routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner })
	if !ok || recipe.MediaPlan == nil {
		return "", nil
	}
	planner := routeIO.MediaPlanner()
	if planner == nil {
		return "", nil
	}
	plan := recipe.MediaPlan(targets)
	if plan == nil {
		return "", nil
	}
	return planner.Apply(ctx, *plan)
}
