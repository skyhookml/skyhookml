package exec_ops

import (
	"../skyhook"

	"fmt"
	"net/http"
	urllib "net/url"
)

func ParentsToDatasets(url string, parents []skyhook.ExecParent) ([]skyhook.Dataset, error) {
	datasets := make([]skyhook.Dataset, len(parents))
	for i, parent := range parents {
		if parent.Type == "n" {
			var parentDatasets []*skyhook.Dataset
			err := skyhook.JsonGet(url, fmt.Sprintf("/exec-nodes/%d/datasets", parent.ID), &parentDatasets)
			if err != nil {
				return nil, fmt.Errorf("error getting datasets of parent node %d: %v", parent.ID, err)
			}
			if parentDatasets[parent.Index] == nil {
				return nil, fmt.Errorf("parent node %d missing dataset at index %d", parent.ID, parent.Index)
			}
			datasets[i] = *parentDatasets[parent.Index]
		} else if parent.Type == "d" {
			var dataset skyhook.Dataset
			err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", parent.ID), &dataset)
			if err != nil {
				return nil, fmt.Errorf("error getting parent dataset %d: %v", parent.ID, err)
			}
			datasets[i] = dataset
		}
	}
	return datasets, nil
}

func GetParentDatasets(url string, node skyhook.ExecNode) ([]skyhook.Dataset, error) {
	return ParentsToDatasets(url, node.Parents)
}

func GetDatasets(url string, ids []int) ([]skyhook.Dataset, error) {
	var datasets []skyhook.Dataset
	for _, id := range ids {
		var dataset skyhook.Dataset
		err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", id), &dataset)
		if err != nil {
			return nil, fmt.Errorf("error getting dataset %d: %v", id, err)
		}
		datasets = append(datasets, dataset)
	}
	return datasets, nil
}

func GetKeys(url string, node skyhook.ExecNode) (map[string]bool, error) {
	datasets, err := ParentsToDatasets(url, node.Parents)
	if err != nil {
		return nil, fmt.Errorf("error getting parent datasets: %v", err)
	}
	items, err := GetItems(url, datasets)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool)
	for key := range items {
		keys[key] = true
	}
	return keys, nil
}

func GetItems(url string, datasets []skyhook.Dataset) (map[string][]skyhook.Item, error) {
	// fetch items
	items := make([]map[string]skyhook.Item, len(datasets))
	for i, dataset := range datasets {
		var curItems []skyhook.Item
		err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), &curItems)
		if err != nil {
			return nil, fmt.Errorf("error getting items in dataset %d: %v", dataset.ID, err)
		}
		items[i] = make(map[string]skyhook.Item)
		for _, item := range curItems {
			items[i][item.Key] = item
		}
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

// make tasks by grouping items of same key across the datasets
func SimpleTasks(url string, node skyhook.ExecNode, rawItems [][]skyhook.Item) []skyhook.ExecTask {
	// group items by key
	items := make([]map[string]skyhook.Item, len(rawItems))
	for i, curItems := range rawItems {
		items[i] = make(map[string]skyhook.Item)
		for _, item := range curItems {
			items[i][item.Key] = item
		}
	}

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
		groupedItems[key] = make([]skyhook.Item, len(items))
		for i := 0; i < len(items); i++ {
			groupedItems[key][i] = items[i][key]
		}
	}

	var tasks []skyhook.ExecTask
	for key, curItems := range groupedItems {
		tasks = append(tasks, skyhook.ExecTask{
			Key: key,
			Items: curItems,
		})
	}
	return tasks
}

func WriteItem(url string, dataset skyhook.Dataset, key string, data skyhook.Data) error {
	ext, format := data.GetDefaultExtAndFormat()
	var item skyhook.Item
	resp, err := http.PostForm(url + fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
		"key": {key},
		"ext": {ext},
		"format": {format},
		"metadata": {string(skyhook.JsonMarshal(data.GetMetadata()))},
	})
	if err != nil {
		return err
	}
	if err := skyhook.ParseJsonResponse(resp, &item); err != nil {
		return err
	}
	item.UpdateData(data)
	return nil
}

func MapGetOutputKeys(node skyhook.ExecNode, inputs [][]string) []string {
	// get shared keys across parents
	keys := make(map[string]bool)
	for _, key := range inputs[0] {
		keys[key] = true
	}
	for _, cur := range inputs[1:] {
		curSet := make(map[string]bool)
		for _, key := range cur {
			curSet[key] = true
		}
		for key := range keys {
			if !curSet[key] {
				delete(keys, key)
			}
		}
	}
	var l []string
	for key := range keys {
		l = append(l, key)
	}
	return l
}

func MapGetNeededInputs(node skyhook.ExecNode, outputs []string) [][]string {
	inputs := make([][]string, len(node.Parents))
	for i := range inputs {
		inputs[i] = outputs
	}
	return inputs
}
