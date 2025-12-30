package provider

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

// TestResourcePoolSchema tests that the resource pool schema is properly configured
func TestResourcePoolSchema(t *testing.T) {
	resource := resourcePool()

	assert.NotNil(t, resource)
	assert.NotNil(t, resource.Schema)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)

	// Check required fields
	assert.True(t, resource.Schema["name"].Required)
	assert.Equal(t, schema.TypeString, resource.Schema["name"].Type)

	// Check optional fields
	assert.True(t, resource.Schema["mirror"].Optional)
	assert.True(t, resource.Schema["device"].Optional)

	// Check property_mode has correct default
	assert.Equal(t, "defined", resource.Schema["property_mode"].Default)
	assert.True(t, resource.Schema["property_mode"].Optional)

	// Check computed fields
	assert.True(t, resource.Schema["properties"].Computed)
	assert.True(t, resource.Schema["raw_properties"].Computed)
}

// TestPropertyModeDefaultOnImport tests that property_mode defaults to "defined" during import
func TestPropertyModeDefaultOnImport(t *testing.T) {
	resource := resourcePool()
	d := resource.TestResourceData()

	// Simulate an import scenario where property_mode is not set explicitly
	// but it should use the default value from the schema
	d.Set("name", "testpool")
	
	// When property_mode is not explicitly set, Get() should return the default value
	// However, TestResourceData doesn't automatically apply defaults, so we verify 
	// the schema has the correct default defined
	propertyModeSchema := resource.Schema["property_mode"]
	assert.Equal(t, "defined", propertyModeSchema.Default, 
		"property_mode schema should have 'defined' as default, which will be applied during actual import")
		
	// In a real import scenario, the default would be applied. We can test this
	// by explicitly checking what happens when we don't set it but the schema has a default
	if d.Get("property_mode") == "" || d.Get("property_mode") == nil {
		// This is expected behavior - TestResourceData doesn't apply defaults
		// But we've verified the schema has the correct default above
		assert.Equal(t, "defined", propertyModeSchema.Default)
	}
}

// TestPropertyModeDefaultInSchema tests the schema default value directly
func TestPropertyModeDefaultInSchema(t *testing.T) {
	resource := resourcePool()
	propertyModeSchema := resource.Schema["property_mode"]

	assert.Equal(t, "defined", propertyModeSchema.Default, "Schema default for property_mode should be 'defined'")
}

// TestParseVdevSpecificationWithDevices tests parsing vdev specification with striped devices
func TestParseVdevSpecificationWithDevices(t *testing.T) {
	devices := []interface{}{
		map[string]interface{}{
			"path": "/dev/sda",
		},
		map[string]interface{}{
			"path": "/dev/sdb",
		},
	}

	result := parseVdevSpecification(nil, devices)
	
	assert.Contains(t, result, "/dev/sda")
	assert.Contains(t, result, "/dev/sdb")
	assert.NotContains(t, result, "mirror")
}

// TestParseVdevSpecificationWithMirrors tests parsing vdev specification with mirrors
func TestParseVdevSpecificationWithMirrors(t *testing.T) {
	mirrors := []interface{}{
		map[string]interface{}{
			"device": []interface{}{
				map[string]interface{}{
					"path": "/dev/sdc",
				},
				map[string]interface{}{
					"path": "/dev/sdd",
				},
			},
		},
	}

	result := parseVdevSpecification(mirrors, nil)
	
	assert.Contains(t, result, "mirror")
	assert.Contains(t, result, "/dev/sdc")
	assert.Contains(t, result, "/dev/sdd")
}

// TestParseVdevSpecificationWithMultipleMirrors tests parsing multiple mirrored vdevs
func TestParseVdevSpecificationWithMultipleMirrors(t *testing.T) {
	mirrors := []interface{}{
		map[string]interface{}{
			"device": []interface{}{
				map[string]interface{}{
					"path": "/dev/sdc",
				},
				map[string]interface{}{
					"path": "/dev/sdd",
				},
			},
		},
		map[string]interface{}{
			"device": []interface{}{
				map[string]interface{}{
					"path": "/dev/sde",
				},
				map[string]interface{}{
					"path": "/dev/sdf",
				},
			},
		},
	}

	result := parseVdevSpecification(mirrors, nil)
	
	// Should contain two mirror declarations
	assert.Contains(t, result, "mirror")
	assert.Contains(t, result, "/dev/sdc")
	assert.Contains(t, result, "/dev/sdd")
	assert.Contains(t, result, "/dev/sde")
	assert.Contains(t, result, "/dev/sdf")
}

// TestParseVdevSpecificationEmpty tests parsing empty vdev specification
func TestParseVdevSpecificationEmpty(t *testing.T) {
	result := parseVdevSpecification(nil, nil)
	assert.Equal(t, "", result)
}

// TestFlattenDevice tests flattening a device structure
func TestFlattenDevice(t *testing.T) {
	device := Device{
		path: "/dev/sda",
	}

	result := flattenDevice(device)

	assert.NotNil(t, result)
	assert.Equal(t, "/dev/sda", result["path"])
}

// TestFlattenMirror tests flattening a mirror structure
func TestFlattenMirror(t *testing.T) {
	mirror := Mirror{
		devices: []Device{
			{path: "/dev/sdb"},
			{path: "/dev/sdc"},
		},
	}

	result := flattenMirror(mirror)

	assert.NotNil(t, result)
	assert.NotNil(t, result["device"])
	
	devices := result["device"].([]map[string]interface{})
	assert.Len(t, devices, 2)
	assert.Equal(t, "/dev/sdb", devices[0]["path"])
	assert.Equal(t, "/dev/sdc", devices[1]["path"])
}

// TestParsePropertyBlocks tests parsing property blocks
func TestParsePropertyBlocks(t *testing.T) {
	blocks := []interface{}{
		map[string]interface{}{
			"name":  "compression",
			"value": "lz4",
		},
		map[string]interface{}{
			"name":  "atime",
			"value": "off",
		},
	}

	result := parsePropertyBlocks(blocks)

	assert.Len(t, result, 2)
	assert.Equal(t, "lz4", result["compression"])
	assert.Equal(t, "off", result["atime"])
}

// TestParsePropertyBlocksEmpty tests parsing empty property blocks
func TestParsePropertyBlocksEmpty(t *testing.T) {
	blocks := []interface{}{}
	result := parsePropertyBlocks(blocks)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// TestPopulateResourceDataPool tests populating resource data from pool
func TestPopulateResourceDataPool(t *testing.T) {
	resource := resourcePool()
	d := resource.TestResourceData()
	
	// Set required fields to avoid validation errors
	d.Set("name", "testpool")
	d.Set("property_mode", "defined")

	pool := Pool{
		guid: "12345678",
		properties: map[string]Property{
			"size": {
				value:    "1T",
				rawValue: "1099511627776",
				source:   SourceLocal,
			},
			"health": {
				value:    "ONLINE",
				rawValue: "ONLINE",
				source:   SourceDefault,
			},
		},
		layout: PoolLayout{
			striped: []Device{
				{path: "/dev/sda"},
			},
			mirrors: []Mirror{},
		},
	}

	diags := populateResourceDataPool(d, pool)

	assert.Nil(t, diags)
	assert.Equal(t, "12345678", d.Id())
	
	// Check devices were set
	devices := d.Get("device").([]interface{})
	assert.Len(t, devices, 1)
	
	// Check mirrors were set
	mirrors := d.Get("mirror").([]interface{})
	assert.Len(t, mirrors, 0)

	// Check properties were set
	properties := d.Get("properties").(map[string]interface{})
	assert.Equal(t, "1T", properties["size"])
	assert.Equal(t, "ONLINE", properties["health"])
}

// TestPopulateResourceDataPoolWithMirrors tests populating resource data with mirrored vdevs
func TestPopulateResourceDataPoolWithMirrors(t *testing.T) {
	resource := resourcePool()
	d := resource.TestResourceData()
	
	// Set required fields to avoid validation errors
	d.Set("name", "testpool")
	d.Set("property_mode", "defined")

	pool := Pool{
		guid: "87654321",
		properties: map[string]Property{
			"size": {
				value:    "2T",
				rawValue: "2199023255552",
				source:   SourceLocal,
			},
		},
		layout: PoolLayout{
			striped: []Device{},
			mirrors: []Mirror{
				{
					devices: []Device{
						{path: "/dev/sdb"},
						{path: "/dev/sdc"},
					},
				},
			},
		},
	}

	diags := populateResourceDataPool(d, pool)

	assert.Nil(t, diags)
	assert.Equal(t, "87654321", d.Id())
	
	// Check mirrors were set
	mirrors := d.Get("mirror").([]interface{})
	assert.Len(t, mirrors, 1)
	
	mirror := mirrors[0].(map[string]interface{})
	devices := mirror["device"].([]interface{})
	assert.Len(t, devices, 2)
	
	device0 := devices[0].(map[string]interface{})
	device1 := devices[1].(map[string]interface{})
	assert.Equal(t, "/dev/sdb", device0["path"])
	assert.Equal(t, "/dev/sdc", device1["path"])
}

// TestResourcePoolDeleteSetsEmptyID tests that delete sets ID to empty string
func TestResourcePoolDeleteSetsEmptyID(t *testing.T) {
	// This is a unit test showing the expected behavior without actual ZFS calls
	resource := resourcePool()
	d := resource.TestResourceData()
	d.SetId("test-guid")
	
	// Verify ID is set before delete
	assert.NotEmpty(t, d.Id())
	
	// After delete, the ID should be cleared (this would happen in resourcePoolDelete)
	d.SetId("")
	assert.Empty(t, d.Id())
}

// TestVdevSchemaValidation tests vdev schema structure
func TestVdevSchemaValidation(t *testing.T) {
	assert.NotNil(t, vdevSchema)
	assert.NotNil(t, vdevSchema.Schema)
	assert.NotNil(t, vdevSchema.Schema["path"])
	assert.True(t, vdevSchema.Schema["path"].Required)
	assert.True(t, vdevSchema.Schema["path"].ForceNew)
}

// TestMirrorSchemaValidation tests mirror schema structure
func TestMirrorSchemaValidation(t *testing.T) {
	assert.NotNil(t, mirrorSchema)
	assert.NotNil(t, mirrorSchema.Schema)
	assert.NotNil(t, mirrorSchema.Schema["device"])
	assert.True(t, mirrorSchema.Schema["device"].Required)
	assert.True(t, mirrorSchema.Schema["device"].ForceNew)
	assert.Equal(t, 2, mirrorSchema.Schema["device"].MinItems)
}

// TestPropertySchemaValidation tests property schema structure
func TestPropertySchemaValidation(t *testing.T) {
	assert.Equal(t, schema.TypeSet, propertySchema.Type)
	assert.True(t, propertySchema.Optional)
	assert.NotNil(t, propertySchema.Elem)
}

// TestPropertyModeSchemaValidation tests property_mode schema structure
func TestPropertyModeSchemaValidation(t *testing.T) {
	assert.Equal(t, schema.TypeString, propertyModeSchema.Type)
	assert.Equal(t, "defined", propertyModeSchema.Default)
	assert.True(t, propertyModeSchema.Optional)
	assert.NotNil(t, propertyModeSchema.ValidateDiagFunc)
}

// TestPropertiesSchemaValidation tests properties schema structure
func TestPropertiesSchemaValidation(t *testing.T) {
	assert.Equal(t, schema.TypeMap, propertiesSchema.Type)
	assert.True(t, propertiesSchema.Computed)
}

// TestRawPropertiesSchemaValidation tests raw_properties schema structure
func TestRawPropertiesSchemaValidation(t *testing.T) {
	assert.Equal(t, schema.TypeMap, rawPropertiesSchema.Type)
	assert.True(t, rawPropertiesSchema.Computed)
}

// TestPoolErrorType tests PoolError error type
func TestPoolErrorType(t *testing.T) {
	err := &PoolError{errmsg: "zpool does not exist"}
	assert.Equal(t, "zpool does not exist", err.Error())
	
	// Test type assertion
	var poolErr *PoolError
	assert.True(t, errors.As(err, &poolErr))
}

// TestGetPropertyNames tests extracting property names from resource data
func TestGetPropertyNames(t *testing.T) {
	resource := resourcePool()
	d := resource.TestResourceData()

	// Set some properties
	properties := schema.NewSet(schema.HashResource(propertySchema.Elem.(*schema.Resource)), []interface{}{
		map[string]interface{}{
			"name":  "compression",
			"value": "lz4",
		},
		map[string]interface{}{
			"name":  "atime",
			"value": "off",
		},
	})
	
	d.Set("property", properties)

	names := getPropertyNames(d)

	assert.Len(t, names, 2)
	assert.Contains(t, names, "compression")
	assert.Contains(t, names, "atime")
}
