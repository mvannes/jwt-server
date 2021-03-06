package jwt

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	validator "github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

func Routes() *chi.Mux {
	h := ProvideJWTHandler()
	r := chi.NewRouter()

	r.Post("/signup", h.SignUpUser)
	r.Post("/signin", h.SigninUser)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/token/invalidate", h.InvalidateToken)

	return r
}

type UserSignUpRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required,max=256"`
	Password string `json:"password" validate:"required"`
}

type UserSignInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserSignInResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type InvalidateTokenRequest struct {
	UUID string `json:"uuid" validate:"required,uuid"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type JWTHandler struct {
	UserRepository UserRepository
	TokenManager   TokenManagerInterface
}

func ProvideJWTHandler() *JWTHandler {
	return &JWTHandler{
		UserRepository: NewJSONUserRepository("users", "people.json"),
		TokenManager:   NewTokenManager(),
	}
}

func (h *JWTHandler) SignUpUser(w http.ResponseWriter, r *http.Request) {
	var ur UserSignUpRequest

	err := json.NewDecoder(r.Body).Decode(&ur)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.New().Struct(ur)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u, err := h.UserRepository.GetUser(ur.Email)
	if nil != err && err != UserNotFoundError {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nil != u {
		http.Error(w, UserExistsError.Error(), http.StatusConflict)
		return
	}

	err = h.UserRepository.StoreUser(ur.Email, ur.Name, ur.Password)
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.PlainText(w, r, "User added successfully")
}

func (h *JWTHandler) SigninUser(w http.ResponseWriter, r *http.Request) {
	var ur UserSignInRequest

	err := json.NewDecoder(r.Body).Decode(&ur)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.New().Struct(ur)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, err := h.UserRepository.GetUser(ur.Email)
	if nil != err {
		if err == UserNotFoundError {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(ur.Password)); nil != err {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	at, err := h.TokenManager.CreateAccessToken(u.Email, u.Name)
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rt, err := h.TokenManager.CreateRefreshToken(u.Email)

	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, UserSignInResponse{
		AccessToken:  at,
		RefreshToken: rt,
	})
}

func (h *JWTHandler) InvalidateToken(w http.ResponseWriter, r *http.Request) {
	var itr InvalidateTokenRequest

	err := json.NewDecoder(r.Body).Decode(&itr)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.New().Struct(itr)

	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.TokenManager.InvalidateRefreshToken(itr.UUID)
	if nil != err {
		if err == errTokenNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.PlainText(w, r, "Token invalidated")
}

func (h *JWTHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {

	var rtr RefreshTokenRequest

	err := json.NewDecoder(r.Body).Decode(&rtr)
	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.New().Struct(rtr)
	if nil != err {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rt, err := h.TokenManager.DecodeRefreshToken(rtr.RefreshToken)
	if nil != err {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	u, err := h.UserRepository.GetUser(rt.Username)
	if nil != err {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	at, err := h.TokenManager.CreateAccessToken(u.Email, u.Name)
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.PlainText(w, r, at)
}
