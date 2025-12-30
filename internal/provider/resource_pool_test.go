package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

// Mock functions for ZFS operations
func mockZfsDescribePool(config *Config, poolName string, requiredProperties []string) (*Pool, error) {
	return &Pool{
		guid: "test-guid-12345",
		properties: map[string]Property{
			"size": {
				value:    "100G",
				rawValue: "107374182400",
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
				{path: "/dev/disk/by-id/test-device"},
			},
			mirrors: []Mirror{},
		},
	}, nil
}

func mockZfsCreatePool(config *Config, pool *CreatePool) (*Pool, error) {
	return &Pool{
		guid: "test-guid-12345",
		properties: map[string]Property{
			"size": {
				value:    "100G",
				rawValue: "107374182400",
				source:   SourceLocal,
			},
		},
		layout: PoolLayout{
			striped: []Device{
				{path: "/dev/disk/by-id/test-device"},
			},
			mirrors: []Mirror{},
		},
	}, nil
}

func mockZfsDestroyPool(config *Config, poolName string) error {
	return nil
}

func mockZfsRenamePool(config *Config, oldName string, newName string) error {
	return nil
}

func mockZfsGetPoolNameByGuid(config *Config, guid string) (*string, error) {
	name := "testpool"
	return &name, nil
}

// setupMockZfs is a helper function to set up mock ZFS functions and return a cleanup function
func setupMockZfs(t *testing.T) {
	t.Helper()

	// Save original functions
	origDescribe := zfsDescribePool
	origCreate := zfsCreatePool
	origDestroy := zfsDestroyPool
	origRename := zfsRenamePool
	origGetName := zfsGetPoolNameByGuid

	// Set mock functions
	zfsDescribePool = mockZfsDescribePool
	zfsCreatePool = mockZfsCreatePool
	zfsDestroyPool = mockZfsDestroyPool
	zfsRenamePool = mockZfsRenamePool
	zfsGetPoolNameByGuid = mockZfsGetPoolNameByGuid

	// Restore original functions after test
	t.Cleanup(func() {
		zfsDescribePool = origDescribe
		zfsCreatePool = origCreate
		zfsDestroyPool = origDestroy
		zfsRenamePool = origRename
		zfsGetPoolNameByGuid = origGetName
	})
}

// TestAccResourcePool_DefaultsToDefinedWhenUnset tests that property_mode defaults to "defined" when not supplied
func TestAccResourcePool_DefaultsToDefinedWhenUnset(t *testing.T) {
	t.Helper()
	setupMockZfs(t)

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePoolDefaults,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zfs_pool.test", "property_mode", "defined"),
					resource.TestCheckResourceAttr("zfs_pool.test", "name", "testpool"),
				),
			},
		},
	})
}

// TestAccResourcePool_Basic tests basic pool creation
func TestAccResourcePool_Basic(t *testing.T) {
	t.Helper()
	setupMockZfs(t)

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePoolBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zfs_pool.test", "name", "testpool"),
					resource.TestCheckResourceAttrSet("zfs_pool.test", "id"),
				),
			},
		},
	})
}

const testAccResourcePoolDefaults = `
resource "zfs_pool" "test" {
  name = "testpool"
  device {
    path = "/dev/disk/by-id/test-device"
  }
}
`

const testAccResourcePoolBasic = `
resource "zfs_pool" "test" {
  name = "testpool"
  device {
    path = "/dev/disk/by-id/test-device"
  }
  property {
    name  = "comment"
    value = "test pool"
  }
}
`

// Unit tests for helper functions
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

// TestPropertyModeDefaultInSchema tests the schema default value directly
func TestPropertyModeDefaultInSchema(t *testing.T) {
	resource := resourcePool()
	propertyModeSchema := resource.Schema["property_mode"]

	assert.Equal(t, "defined", propertyModeSchema.Default, "Schema default for property_mode should be 'defined'")
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
