/*
Membership API

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package membershipclient

import (
	"encoding/json"
)

// checks if the StackUserAccessAllOf type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &StackUserAccessAllOf{}

// StackUserAccessAllOf struct for StackUserAccessAllOf
type StackUserAccessAllOf struct {
	// Stack ID
	StackId string `json:"stackId"`
	// User ID
	UserId string `json:"userId"`
	// User email
	Email string `json:"email"`
	Role Role `json:"role"`
}

// NewStackUserAccessAllOf instantiates a new StackUserAccessAllOf object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewStackUserAccessAllOf(stackId string, userId string, email string, role Role) *StackUserAccessAllOf {
	this := StackUserAccessAllOf{}
	this.StackId = stackId
	this.UserId = userId
	this.Email = email
	this.Role = role
	return &this
}

// NewStackUserAccessAllOfWithDefaults instantiates a new StackUserAccessAllOf object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewStackUserAccessAllOfWithDefaults() *StackUserAccessAllOf {
	this := StackUserAccessAllOf{}
	return &this
}

// GetStackId returns the StackId field value
func (o *StackUserAccessAllOf) GetStackId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.StackId
}

// GetStackIdOk returns a tuple with the StackId field value
// and a boolean to check if the value has been set.
func (o *StackUserAccessAllOf) GetStackIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.StackId, true
}

// SetStackId sets field value
func (o *StackUserAccessAllOf) SetStackId(v string) {
	o.StackId = v
}

// GetUserId returns the UserId field value
func (o *StackUserAccessAllOf) GetUserId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.UserId
}

// GetUserIdOk returns a tuple with the UserId field value
// and a boolean to check if the value has been set.
func (o *StackUserAccessAllOf) GetUserIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.UserId, true
}

// SetUserId sets field value
func (o *StackUserAccessAllOf) SetUserId(v string) {
	o.UserId = v
}

// GetEmail returns the Email field value
func (o *StackUserAccessAllOf) GetEmail() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Email
}

// GetEmailOk returns a tuple with the Email field value
// and a boolean to check if the value has been set.
func (o *StackUserAccessAllOf) GetEmailOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Email, true
}

// SetEmail sets field value
func (o *StackUserAccessAllOf) SetEmail(v string) {
	o.Email = v
}

// GetRole returns the Role field value
func (o *StackUserAccessAllOf) GetRole() Role {
	if o == nil {
		var ret Role
		return ret
	}

	return o.Role
}

// GetRoleOk returns a tuple with the Role field value
// and a boolean to check if the value has been set.
func (o *StackUserAccessAllOf) GetRoleOk() (*Role, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Role, true
}

// SetRole sets field value
func (o *StackUserAccessAllOf) SetRole(v Role) {
	o.Role = v
}

func (o StackUserAccessAllOf) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o StackUserAccessAllOf) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["stackId"] = o.StackId
	toSerialize["userId"] = o.UserId
	toSerialize["email"] = o.Email
	toSerialize["role"] = o.Role
	return toSerialize, nil
}

type NullableStackUserAccessAllOf struct {
	value *StackUserAccessAllOf
	isSet bool
}

func (v NullableStackUserAccessAllOf) Get() *StackUserAccessAllOf {
	return v.value
}

func (v *NullableStackUserAccessAllOf) Set(val *StackUserAccessAllOf) {
	v.value = val
	v.isSet = true
}

func (v NullableStackUserAccessAllOf) IsSet() bool {
	return v.isSet
}

func (v *NullableStackUserAccessAllOf) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableStackUserAccessAllOf(val *StackUserAccessAllOf) *NullableStackUserAccessAllOf {
	return &NullableStackUserAccessAllOf{value: val, isSet: true}
}

func (v NullableStackUserAccessAllOf) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableStackUserAccessAllOf) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


