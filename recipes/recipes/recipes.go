package recipes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Recipe represents a single recipe loaded from a JSON file.
type Recipe struct {
	Name            string       `json:"name"`
	Slug            string       `json:"slug"`
	Description     string       `json:"description"`
	Kind            string       `json:"kind"`
	CookTimeMinutes int          `json:"cookTimeMinutes"`
	Instructions    []string     `json:"instructions"`
	Ingredients     []Ingredient `json:"ingredients"`
}

// Ingredient represents a single ingredient in a recipe.
type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string into a URL-friendly slug.
// For example, "Pasta Carbonara" becomes "pasta-carbonara".
func Slugify(s string) string {
	slug := strings.ToLower(strings.TrimSpace(s))
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// LoadRecipe reads a single recipe JSON file from the given directory by slug.
// The filename is expected to be <slug>.json.
func LoadRecipe(dir string, slug string) (Recipe, error) {
	path := filepath.Join(dir, slug+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, fmt.Errorf("reading file %s: %w", path, err)
	}

	var r Recipe
	if err := json.Unmarshal(data, &r); err != nil {
		return Recipe{}, fmt.Errorf("parsing recipe from %s: %w", path, err)
	}

	return r, nil
}

// LoadRecipes reads all JSON files from the given directory and returns
// a slice of Recipe values parsed from those files.
func LoadRecipes(dir string) ([]Recipe, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var recipes []Recipe
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", path, err)
		}

		var r Recipe
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("parsing recipe from %s: %w", path, err)
		}

		recipes = append(recipes, r)
	}

	return recipes, nil
}

// SaveRecipe writes a recipe to a JSON file in the given directory.
// The filename is derived from the recipe's slug.
func SaveRecipe(dir string, r Recipe) error {
	data, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return fmt.Errorf("marshaling recipe: %w", err)
	}

	path := filepath.Join(dir, r.Slug+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}

	return nil
}

// RecipeExists checks whether a recipe with the given slug already exists.
func RecipeExists(dir string, slug string) bool {
	path := filepath.Join(dir, slug+".json")
	_, err := os.Stat(path)
	return err == nil
}

// DeleteRecipe removes a recipe JSON file from the given directory by slug.
func DeleteRecipe(dir string, slug string) error {
	path := filepath.Join(dir, slug+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting file %s: %w", path, err)
	}
	return nil
}
