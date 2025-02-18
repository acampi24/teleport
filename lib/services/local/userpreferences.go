/*
 *
 * Copyright 2023 Gravitational, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package local

import (
	"context"
	"encoding/json"

	"github.com/gravitational/trace"
	"google.golang.org/protobuf/reflect/protoreflect"

	userpreferencesv1 "github.com/gravitational/teleport/api/gen/proto/go/userpreferences/v1"
	"github.com/gravitational/teleport/lib/backend"
)

// UserPreferencesService is responsible for managing a user's preferences.
type UserPreferencesService struct {
	backend.Backend
}

func DefaultUserPreferences() *userpreferencesv1.UserPreferences {
	return &userpreferencesv1.UserPreferences{
		Assist: &userpreferencesv1.AssistUserPreferences{
			PreferredLogins: []string{},
			ViewMode:        userpreferencesv1.AssistViewMode_ASSIST_VIEW_MODE_DOCKED,
		},
		Theme: userpreferencesv1.Theme_THEME_LIGHT,
		Onboard: &userpreferencesv1.OnboardUserPreferences{
			PreferredResources: []userpreferencesv1.Resource{},
			MarketingParams:    &userpreferencesv1.MarketingParams{},
		},
	}
}

// NewUserPreferencesService returns a new instance of the UserPreferencesService.
func NewUserPreferencesService(backend backend.Backend) *UserPreferencesService {
	return &UserPreferencesService{
		Backend: backend,
	}
}

// GetUserPreferences returns the user preferences for the given user.
func (u *UserPreferencesService) GetUserPreferences(ctx context.Context, username string) (*userpreferencesv1.UserPreferences, error) {
	preferences, err := u.getUserPreferences(ctx, username)
	if err != nil {
		if trace.IsNotFound(err) {
			return DefaultUserPreferences(), nil
		}

		return nil, trace.Wrap(err)
	}

	return preferences, nil
}

// UpsertUserPreferences creates or updates user preferences for a given username.
func (u *UserPreferencesService) UpsertUserPreferences(ctx context.Context, username string, prefs *userpreferencesv1.UserPreferences) error {
	if username == "" {
		return trace.BadParameter("missing username")
	}
	if err := validatePreferences(prefs); err != nil {
		return trace.Wrap(err)
	}

	preferences, err := u.getUserPreferences(ctx, username)
	if err != nil {
		if !trace.IsNotFound(err) {
			return trace.Wrap(err)
		}
		preferences = DefaultUserPreferences()
	}

	if err := overwriteValues(preferences, prefs); err != nil {
		return trace.Wrap(err)
	}

	item, err := createBackendItem(username, preferences)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = u.Put(ctx, item)

	return trace.Wrap(err)
}

// getUserPreferences returns the user preferences for the given username.
func (u *UserPreferencesService) getUserPreferences(ctx context.Context, username string) (*userpreferencesv1.UserPreferences, error) {
	existing, err := u.Get(ctx, backendKey(username))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var p userpreferencesv1.UserPreferences
	if err := json.Unmarshal(existing.Value, &p); err != nil {
		return nil, trace.Wrap(err)
	}

	// Apply the default values to the existing preferences.
	// This allows updating the preferences schema without returning empty values
	// for new fields in the existing preferences.
	df := DefaultUserPreferences()
	if err := overwriteValues(df, &p); err != nil {
		return nil, trace.Wrap(err)
	}

	return df, nil
}

// backendKey returns the backend key for the user preferences for the given username.
func backendKey(username string) []byte {
	return backend.Key(userPreferencesPrefix, username)
}

// validatePreferences validates the given preferences.
func validatePreferences(preferences *userpreferencesv1.UserPreferences) error {
	if preferences == nil {
		return trace.BadParameter("missing preferences")
	}

	return nil
}

// createBackendItem creates a backend.Item for the given username and user preferences.
func createBackendItem(username string, preferences *userpreferencesv1.UserPreferences) (backend.Item, error) {
	settingsKey := backend.Key(userPreferencesPrefix, username)

	payload, err := json.Marshal(preferences)
	if err != nil {
		return backend.Item{}, trace.Wrap(err)
	}

	item := backend.Item{
		Key:   settingsKey,
		Value: payload,
	}

	return item, nil
}

// overwriteValues overwrites the values in dst with the values in src.
// This function uses proto.Ranges internally to iterate over the fields in src.
// Because of this, only non-nil/empty fields in src will overwrite the values in dst.
func overwriteValues(dst, src protoreflect.ProtoMessage) error {
	d := dst.ProtoReflect()
	s := src.ProtoReflect()

	dName := d.Descriptor().FullName().Name()
	sName := s.Descriptor().FullName().Name()
	// If the names don't match, then the types don't match, so we can't overwrite.
	if dName != sName {
		return trace.BadParameter("dst and src must be the same type")
	}

	overwriteValuesRecursive(d, s)

	return nil
}

// overwriteValuesRecursive recursively overwrites the values in dst with the values in src.
// It's a helper function for overwriteValues.
func overwriteValuesRecursive(dst, src protoreflect.Message) {
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.Message() != nil:
			overwriteValuesRecursive(dst.Mutable(fd).Message(), src.Get(fd).Message())
		default:
			dst.Set(fd, src.Get(fd))
		}

		return true
	})
}
