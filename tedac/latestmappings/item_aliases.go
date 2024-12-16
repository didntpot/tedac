package latestmappings

import (
	_ "embed"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var (
	//go:embed item_aliases.nbt
	itemAliasesData []byte

	// aliasMappings maps from a legacy item name alias to an updated name.
	itemAliasMappings = map[string]string{}
	// reverseAliasMappings maps from an updated item name to a legacy item name alias.
	reverseItemAliasMappings = map[string]string{}
)

// UpdatedItemNameFromAlias returns the updated name of an item from a legacy alias. If no alias was found, the
// second return value will be false.
func UpdatedItemNameFromAlias(name string) (string, bool) {
	if updated, ok := itemAliasMappings[name]; ok {
		return updated, true
	}
	return name, false
}

// AliasFromUpdatedItemName returns the legacy alias of an item from an updated name. If no alias was found, the
// second return value will be false.
func AliasFromUpdatedItemName(name string) (string, bool) {
	if alias, ok := reverseItemAliasMappings[name]; ok {
		return alias, true
	}
	return name, false
}

// init creates conversions for each legacy and alias entry.
func init() {
	if err := nbt.UnmarshalEncoding(itemAliasesData, &itemAliasMappings, nbt.BigEndian); err != nil {
		panic(err)
	}
	for name, alias := range itemAliasMappings {
		// have to append "minecraft:", because fuck you cadet
		itemAliasMappings["minecraft:"+name] = "minecraft:" + alias
	}
	for alias, name := range itemAliasMappings {
		reverseItemAliasMappings[name] = alias
	}
}
