// package main

// import (
// 	"fmt"
// 	"log"

// 	"github.com/k8sgpt-ai/k8sgpt/pkg/api"
// 	"github.com/spf13/viper"
// )

// func init() {
// 	// Initialize viper configuration
// 	viper.SetConfigName("k8sgpt")
// 	viper.SetConfigType("yaml")
	
// 	// Use XDG config paths (based on the changelog mentioning XDG conform location)
// 	viper.AddConfigPath("$HOME/.config/k8sgpt")
// 	viper.AddConfigPath(".")
	
// 	// Create config if it doesn't exist
// 	if err := viper.ReadInConfig(); err != nil {
// 		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
// 			// Config file not found, create it
// 			viper.SetConfigFile("$HOME/.config/k8sgpt/k8sgpt.yaml")
// 			viper.Set("ai", map[string]interface{}{
// 				"providers": []interface{}{},
// 			})
// 			if err := viper.WriteConfig(); err != nil {
// 				log.Fatalf("Error creating config file: %v", err)
// 			}
// 		} else {
// 			// Config file found but another error occurred
// 			log.Fatalf("Error reading config file: %v", err)
// 		}
// 	}
// }

// func main() {
// 	// Add OpenAI provider
// 	err := api.AddAuthProvider(api.AddAuthProviderOptions{
// 		Backend:     "openai",
// 		Model:       "gpt-4o",
// 		Password:    "your-api-key-here", // Replace with your actual API key
// 		Temperature: 0.7,
// 		TopP:        0.5,
// 		TopK:        50,
// 		MaxTokens:   2048,
// 	})

// 	if err != nil {
// 		log.Fatalf("Failed to add auth provider: %v", err)
// 	}

// 	fmt.Println("Successfully added OpenAI provider")
// }
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/k8sgpt-ai/k8sgpt/pkg/ai"
	"github.com/k8sgpt-ai/k8sgpt/pkg/api"
	"github.com/spf13/viper"
)

func init() {
	// Initialize viper configuration
	viper.SetConfigName("k8sgpt")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/k8sgpt")
	viper.AddConfigPath(".")
	
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			viper.SetConfigFile("/home/ubuntu/.kube/config")
			viper.Set("ai", map[string]interface{}{
				"providers": []interface{}{},
			})
			if err := viper.WriteConfig(); err != nil {
				log.Fatalf("Error creating config file: %v", err)
			}
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}
}

func main() {
	// Add provider handler
	http.HandleFunc("/api/auth/provider", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addProviderHandler(w, r)
		case http.MethodGet:
			listProvidersHandler(w, r)
		case http.MethodDelete:
			removeProviderHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Set default provider handler
	http.HandleFunc("/api/auth/default", setDefaultProviderHandler)

	

	// Start server
	port := 9090 // Using a different port than k8sgpt serve
	fmt.Printf("Starting API server on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func addProviderHandler(w http.ResponseWriter, r *http.Request) {
	var opts api.AddAuthProviderOptions
	err := json.NewDecoder(r.Body).Decode(&opts)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = api.AddAuthProvider(opts)
	if err != nil {
		http.Error(w, "Failed to add auth provider: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Auth provider added successfully",
	})
}

func listProvidersHandler(w http.ResponseWriter, r *http.Request) {
	var configAI struct {
		Providers []ai.AIProvider `json:"providers"`
	}
	
	err := viper.UnmarshalKey("ai", &configAI)
	if err != nil {
		http.Error(w, "Error reading configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configAI.Providers)
}

func removeProviderHandler(w http.ResponseWriter, r *http.Request) {
	backend := r.URL.Query().Get("backend")
	if backend == "" {
		http.Error(w, "Backend parameter is required", http.StatusBadRequest)
		return
	}

	var configAI struct {
		Providers []ai.AIProvider `json:"providers"`
	}
	
	err := viper.UnmarshalKey("ai", &configAI)
	if err != nil {
		http.Error(w, "Error reading configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	newProviders := []ai.AIProvider{}
	for _, provider := range configAI.Providers {
		if provider.Name != backend {
			newProviders = append(newProviders, provider)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	configAI.Providers = newProviders
	viper.Set("ai", configAI)
	
	if err := viper.WriteConfig(); err != nil {
		http.Error(w, "Error writing configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Auth provider removed successfully",
	})
}

func setDefaultProviderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Backend string `json:"backend"`
	}
	
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if request.Backend == "" {
		http.Error(w, "Backend is required", http.StatusBadRequest)
		return
	}

	// Check if provider exists
	var configAI struct {
		Providers []ai.AIProvider `json:"providers"`
	}
	
	err = viper.UnmarshalKey("ai", &configAI)
	if err != nil {
		http.Error(w, "Error reading configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, provider := range configAI.Providers {
		if provider.Name == request.Backend {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	// Set default provider
	viper.Set("ai.default", request.Backend)
	
	if err := viper.WriteConfig(); err != nil {
		http.Error(w, "Error writing configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Default provider set successfully",
	})
}
