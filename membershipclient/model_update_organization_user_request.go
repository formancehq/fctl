/*
Membership API

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package membershipclient

import (
	"encoding/json"
	"fmt"
)

// checks if the UpdateOrganizationUserRequest type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &UpdateOrganizationUserRequest{}

// UpdateOrganizationUserRequest struct for UpdateOrganizationUserRequest
type UpdateOrganizationUserRequest struct {
	Role Role `json:"role"`
	AdditionalProperties map[string]interface{}
}

type _UpdateOrganizationUserRequest UpdateOrganizationUserRequest

// NewUpdateOrganizationUserRequest instantiates a new UpdateOrganizationUserRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewUpdateOrganizationUserRequest(role Role) *UpdateOrganizationUserRequest {
	this := UpdateOrganizationUserRequest{}
	this.Role = role
	return &this
}

// NewUpdateOrganizationUserRequestWithDefaults instantiates a new UpdateOrganizationUserRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewUpdateOrganizationUserRequestWithDefaults() *UpdateOrganizationUserRequest {
	this := UpdateOrganizationUserRequest{}
	return &this
}

// GetRole returns the Role field value
func (o *UpdateOrganizationUserRequest) GetRole() Role {
	if o == nil {
		var ret Role
		return ret
	}

	return o.Role
}

// GetRoleOk returns a tuple with the Role field value
// and a boolean to check if the value has been set.
func (o *UpdateOrganizationUserRequest) GetRoleOk() (*Role, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Role, true
}

// SetRole sets field value
func (o *UpdateOrganizationUserRequest) SetRole(v Role) {
	o.Role = v
}

func (o UpdateOrganizationUserRequest) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o UpdateOrganizationUserRequest) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["role"] = o.Role

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}

	return toSerialize, nil
}

func (o *UpdateOrganizationUserRequest) UnmarshalJSON(data []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"role",
	}

	allProperties := make(map[string]interface{})

	err = json.Unmarshal(data, &allProperties)

	if err != nil {
		return err;
	}

	for _, requiredProperty := range(requiredProperties) {
		if _, exists := allProperties[requiredProperty]; !exists {
			return fmt.Errorf("no value given for required property %v", requiredProperty)
		}
	}

	varUpdateOrganizationUserRequest := _UpdateOrganizationUserRequest{}

	err = json.Unmarshal(data, &varUpdateOrganizationUserRequest)

	if err != nil {
		return err
	}

	*o = UpdateOrganizationUserRequest(varUpdateOrganizationUserRequest)

	additionalProperties := make(map[string]interface{})

	if err = json.Unmarshal(data, &additionalProperties); err == nil {
		delete(additionalProperties, "role")
		o.AdditionalProperties = additionalProperties
	}

	return err
}

type NullableUpdateOrganizationUserRequest struct {
	value *UpdateOrganizationUserRequest
	isSet bool
}

func (v NullableUpdateOrganizationUserRequest) Get() *UpdateOrganizationUserRequest {
	return v.value
}

func (v *NullableUpdateOrganizationUserRequest) Set(val *UpdateOrganizationUserRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableUpdateOrganizationUserRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableUpdateOrganizationUserRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableUpdateOrganizationUserRequest(val *UpdateOrganizationUserRequest) *NullableUpdateOrganizationUserRequest {
	return &NullableUpdateOrganizationUserRequest{value: val, isSet: true}
}

func (v NullableUpdateOrganizationUserRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableUpdateOrganizationUserRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


