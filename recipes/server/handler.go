package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jesperkha/recipes/recipes"
)

var templateFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"boldIngredients": func(text string, ingredients []recipes.Ingredient) template.HTML {
		escaped := template.HTMLEscapeString(text)
		for _, ing := range ingredients {
			name := ing.Name
			lower := strings.ToLower(escaped)
			target := strings.ToLower(name)
			var result strings.Builder
			pos := 0
			for {
				idx := strings.Index(lower[pos:], target)
				if idx == -1 {
					result.WriteString(escaped[pos:])
					break
				}
				result.WriteString(escaped[pos : pos+idx])
				result.WriteString("<strong>")
				result.WriteString(escaped[pos+idx : pos+idx+len(target)])
				result.WriteString("</strong>")
				pos += idx + len(target)
			}
			escaped = result.String()
		}
		return template.HTML(escaped)
	},
}

func pingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ping"))
	}
}

func recipeHandler() http.HandlerFunc {
	tmpl := template.Must(template.New("recipe.html").Funcs(templateFuncs).ParseFiles("web/recipe.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "name")

		recipe, err := recipes.LoadRecipe("data", slug)
		if err != nil {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, recipe); err != nil {
			http.Error(w, "failed to render template", http.StatusInternalServerError)
		}
	}
}

func recipeAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "name")

		recipe, err := recipes.LoadRecipe("data", slug)
		if err != nil {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipe)
	}
}

func createHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/create.html")
	}
}

func checkPassword(password, hash string) bool {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:]) == strings.ToLower(hash)
}

func authHandler(passwordHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if !checkPassword(body.Password, passwordHash) {
			http.Error(w, "wrong password", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func createRecipeHandler(passwordHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")
		if !checkPassword(password, passwordHash) {
			log.Printf("Got %s", password)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		isEdit := r.FormValue("edit") == "true"
		cookTime, _ := strconv.Atoi(r.FormValue("cookTimeMinutes"))

		names := r.Form["ingredientName"]
		amounts := r.Form["ingredientAmount"]
		units := r.Form["ingredientUnit"]

		var ingredients []recipes.Ingredient
		for i := range names {
			amount, _ := strconv.ParseFloat(amounts[i], 64)
			ingredients = append(ingredients, recipes.Ingredient{
				Name:   names[i],
				Amount: amount,
				Unit:   units[i],
			})
		}

		recipe := recipes.Recipe{
			Name:            r.FormValue("name"),
			Slug:            recipes.Slugify(r.FormValue("name")),
			Description:     r.FormValue("description"),
			Kind:            r.FormValue("kind"),
			CookTimeMinutes: cookTime,
			Instructions:    r.Form["instructions"],
			Ingredients:     ingredients,
		}

		// If not editing, check that the recipe doesn't already exist
		if !isEdit && recipes.RecipeExists("data", recipe.Slug) {
			http.Error(w, "recipe already exists", http.StatusConflict)
			return
		}

		if err := recipes.SaveRecipe("data", recipe); err != nil {
			http.Error(w, "failed to save recipe", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/recipe/"+recipe.Slug, http.StatusSeeOther)
	}
}

func indexHandler() http.HandlerFunc {
	tmpl := template.Must(template.New("index.html").ParseFiles("web/index.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		allRecipes, err := recipes.LoadRecipes("data")
		if err != nil {
			http.Error(w, "failed to load recipes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, allRecipes); err != nil {
			http.Error(w, "failed to render template", http.StatusInternalServerError)
		}
	}
}

func assetsHandler() http.HandlerFunc {
	return http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))).ServeHTTP
}

func deleteRecipeHandler(passwordHash string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if !checkPassword(body.Password, passwordHash) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		slug := chi.URLParam(r, "name")
		if err := recipes.DeleteRecipe("data", slug); err != nil {
			http.Error(w, "failed to delete recipe", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
