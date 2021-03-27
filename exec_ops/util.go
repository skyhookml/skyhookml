package exec_ops

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	urllib "net/url"
)

func GetDataset(url string, id int) (skyhook.Dataset, error) {
	var dataset skyhook.Dataset
	err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", id), &dataset)
	if err != nil {
		return skyhook.Dataset{}, fmt.Errorf("error getting dataset %d: %v", id, err)
	}
	return dataset, nil
}

func GetDatasets(url string, ids []int) ([]skyhook.Dataset, error) {
	var datasets []skyhook.Dataset
	for _, id := range ids {
		dataset, err := GetDataset(url, id)
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, dataset)
	}
	return datasets, nil
}

func GetDatasetItems(url string, dataset skyhook.Dataset) (map[string]skyhook.Item, error) {
	var rawItems []skyhook.Item
	err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), &rawItems)
	if err != nil {
		return nil, fmt.Errorf("error getting items in dataset %d: %v", dataset.ID, err)
	}
	items := make(map[string]skyhook.Item)
	for _, item := range rawItems {
		items[item.Key] = item
	}
	return items, nil
}

func GetItems(url string, datasets []skyhook.Dataset) (map[string][]skyhook.Item, error) {
	// fetch items
	items := make([]map[string]skyhook.Item, len(datasets))
	for i, dataset := range datasets {
		curItems, err := GetDatasetItems(url, dataset)
		if err != nil {
			return nil, err
		}
		items[i] = curItems
	}

	// find shared keys across all datasets
	keys := make(map[string]bool)
	for key := range items[0] {
		keys[key] = true
	}
	for _, curItems := range items[1:] {
		for key := range keys {
			if _, ok := curItems[key]; !ok {
				delete(keys, key)
			}
		}
	}

	groupedItems := make(map[string][]skyhook.Item)
	for key := range keys {
		groupedItems[key] = make([]skyhook.Item, len(datasets))
		for i := 0; i < len(datasets); i++ {
			groupedItems[key][i] = items[i][key]
		}
	}
	return groupedItems, nil
}

// group together items of the same key across the datasets
// rawItems[inp][i][j] is the jth item in the ith dataset for input "inp"
// returns map: key -> input -> list of corresponding items in each dataset
// items with keys that don't appear in all datasets are dropped
func GroupItems(rawItems map[string][][]skyhook.Item) map[string]map[string][]skyhook.Item {
	var numDatasets int = 0
	for _, inputItems := range rawItems {
		numDatasets += len(inputItems)
	}

	keyHits := make(map[string]int)
	// map from (name, key) -> items in each dataset
	itemsByNameKey := make(map[[2]string][]skyhook.Item)
	for name, inputItems := range rawItems {
		for _, curItems := range inputItems {
			keySet := make(map[string]bool)
			for _, item := range curItems {
				if keySet[item.Key] {
					continue
				}
				keySet[item.Key] = true
				k := [2]string{name, item.Key}
				itemsByNameKey[k] = append(itemsByNameKey[k], item)
			}
			for key := range keySet {
				keyHits[key]++
			}
		}
	}

	// only retain keys that appear in all the datasets
	keys := make(map[string]bool)
	for key, hits := range keyHits {
		if hits < numDatasets {
			continue
		}
		keys[key] = true
	}

	items := make(map[string]map[string][]skyhook.Item)
	for key := range keys {
		items[key] = make(map[string][]skyhook.Item)
		for name := range rawItems {
			items[key][name] = itemsByNameKey[[2]string{name, key}]
		}
	}

	return items
}

// make tasks by grouping items of same key across the datasets
func SimpleTasks(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
	groupedItems := GroupItems(rawItems)
	var tasks []skyhook.ExecTask
	for key, curItems := range groupedItems {
		taskItems := make(map[string][][]skyhook.Item)
		for name, itemList := range curItems {
			taskItems[name] = make([][]skyhook.Item, len(itemList))
			for i, item := range itemList {
				taskItems[name][i] = []skyhook.Item{item}
			}
		}
		tasks = append(tasks, skyhook.ExecTask{
			Key: key,
			Items: taskItems,
		})
	}
	return tasks, nil
}

// make a single task with all the input items
func SingleTask(key string) func(skyhook.Runnable, map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
	return func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
		return []skyhook.ExecTask{{
			Key: key,
			Items: rawItems,
		}}, nil
	}
}

func AddItem(url string, dataset skyhook.Dataset, key string, ext string, format string, metadata string) (skyhook.Item, error) {
	var item skyhook.Item
	err := skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
		"key": {key},
		"ext": {ext},
		"format": {format},
		"metadata": {metadata},
	}, &item)
	return item, err
}

func WriteItemWithFormat(url string, dataset skyhook.Dataset, key string, data skyhook.Data, ext string, format string) error {
	metadata := string(skyhook.JsonMarshal(data.GetMetadata()))
	item, err := AddItem(url, dataset, key, ext, format, metadata)
	if err != nil {
		return err
	}
	item.UpdateData(data)
	return nil
}

func WriteItem(url string, dataset skyhook.Dataset, key string, data skyhook.Data) error {
	ext, format := data.GetDefaultExtAndFormat()
	return WriteItemWithFormat(url, dataset, key, data, ext, format)
}

func MapGetOutputKeys(node skyhook.ExecNode, inputs map[string][][]string) []string {
	// get shared keys across parents
	var numDatasets int = 0
	for _, keyLists := range inputs {
		numDatasets += len(keyLists)
	}

	keyHits := make(map[string]int)
	for _, keyLists := range inputs {
		for _, keyList := range keyLists {
			keySet := make(map[string]bool)
			for _, key := range keyList {
				keySet[key] = true
			}
			for key := range keySet {
				keyHits[key]++
			}
		}
	}

	var outputKeys []string
	for key, hits := range keyHits {
		if hits < numDatasets {
			continue
		}
		outputKeys = append(outputKeys, key)
	}
	return outputKeys
}

func MapGetNeededInputs(node skyhook.ExecNode, outputs []string) map[string][][]string {
	// broadcast the output keys over all of the inputs for this node
	needed := make(map[string][][]string)
	for name, plist := range node.Parents {
		needed[name] = make([][]string, len(plist))
		for i := range plist {
			needed[name][i] = outputs
		}
	}
	return needed
}

func GetOutputsSimilarToInputs(params string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
	// output outputs0, outputs1, ... for each dataset in inputs
	var outputs []skyhook.ExecOutput
	for i, inputType := range inputTypes["inputs"] {
		outputs = append(outputs, skyhook.ExecOutput{
			Name: fmt.Sprintf("outputs%d", i),
			DataType: inputType,
		})
	}
	return outputs
}
