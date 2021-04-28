import utils from './utils.js';

// Helper function to get suitable ExecParent options based on datasets and
// nodes in the workspace. It calls the callback with a list of ExecParent
// objects that have an additional Label field indicating the dataset or node
// name.
export default function(ws, comp, callback) {
	let options = [];
	Promise.all([
		utils.request(comp, 'GET', '/datasets', null, (datasets) => {
			for(let ds of datasets) {
				if(ds.Type == 'computed') {
					continue;
				}
				options.push({
					Type: 'd',
					ID: ds.ID,
					DataType: ds.DataType,
					Label: 'Dataset: ' + ds.Name,
				});
			}
		}),
		utils.request(comp, 'GET', '/exec-nodes?ws='+ws, null, (nodes) => {
			for(let node of nodes) {
				for(let output of node.Outputs) {
					options.push({
						Type: 'n',
						ID: node.ID,
						Name: output.Name,
						DataType: output.DataType,
						Label: 'Node: ' + node.Name + '['+output.Name+']',
					});
				}
			}
		}),
	]).then(() => {
		callback(options);
	});
};
