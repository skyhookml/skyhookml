import utils from './utils.js';
import PipelineSelectedPane from './pipeline-selected-pane.js';
import AddExecNode from './add-exec-node.js';

export default {
	components: {
		'pipeline-selected-pane': PipelineSelectedPane,
		'add-exec-node': AddExecNode,
	},
	data: function() {
		return {
			datasets: {},
			nodes: {},
			workspaces: [],
			sorted: [],
			selectedNode: null,
			showingNewNodeModal: false,
		};
	},
	created: function() {
		utils.request(this, 'GET', '/workspaces', null, (data) => {
			this.workspaces = data;
		});
	},
	// don't want to render until after mounted
	mounted: function() {
		this.update();
	},
	methods: {
		update: function() {
			utils.request(this, 'GET', '/datasets', null, (data) => {
				let datasets = {};
				data.forEach((ds) => {
					datasets[ds.ID] = ds;
				});
				this.datasets = datasets;
				this.processNodes();
			});
			utils.request(this, 'GET', '/exec-nodes?ws='+this.$route.params.ws, null, (nodeList) => {
				let nodes = {};
				nodeList.forEach((node) => {
					nodes[node.ID] = node;
				});
				this.nodes = nodes;
				this.processNodes();

				if(this.selectedNode) {
					if(this.nodes[this.selectedNode.ID]) {
						this.selectNode(this.nodes[this.selectedNode.ID]);
					} else {
						this.selectedNode = null;
					}
				}
			});
		},
		processNodes: function() {
			// perform topological sort over ExecNodes so that we can display them in
			// order based on dependencies
			// we implement a simple O(n^2) algorithm here rather than BFS
			let sorted = [];
			let needed = {};
			for(var nodeID in this.nodes) {
				needed[nodeID] = this.nodes[nodeID];
			}
			while(Object.keys(needed).length > 0) {
				for(let nodeID in needed) {
					let node = needed[nodeID];

					// this node is ready if none of its parents appear in needed
					let ready = true;
					for(let plist of Object.values(node.Parents)) {
						for(let parent of plist) {
							if(parent.Type != 'n') {
								continue;
							}
							if(!needed[parent.ID]) {
								continue;
							}
							ready = false;
						}
					}
					if(!ready) {
						continue;
					}
					delete needed[nodeID];

					// create a string summary of the node inputs
					let inputs = [];
					for(let [inputName, plist] of Object.entries(node.Parents)) {
						let curInput = [];
						for(let parent of plist) {
							if(parent.Type == 'd') {
								let dataset = this.datasets[parent.ID];
								if(!dataset) {
									curInput.push('unknown dataset');
									continue;
								}
								curInput.push(dataset.Name);
							} else if(parent.Type == 'n') {
								let n = this.nodes[parent.ID];
								if(!n) {
									curInput.push('unknown node');
									continue;
								}
								curInput.push(n.Name+'['+parent.Name+']');
							}
						}

						if(curInput.length > 1) {
							let summary = curInput.join(', ');
							inputs.push(inputName+'='+'('+summary+')');
						} else if(curInput.length == 1) {
							inputs.push(inputName+'='+curInput[0]);
						} else {
							inputs.push(inputName+'=()')
						}
					}
					let inputSummary = inputs.join(', ');

					// also summarize its outputs
					let outputSummary = node.Outputs.map((output) => {
						return output.Name;
					}).join(', ');

					sorted.push({
						'Node': node,
						'Inputs': inputSummary,
						'Outputs': outputSummary,
					});
				}
			}

			this.sorted = sorted;
		},
		showNewNodeModal: function() {
			// if we're already showing it, we have to force re-create the component
			if(this.showingNewNodeModal) {
				this.showingNewNodeModal = false;
				Vue.nextTick(() => {
					this.showingNewNodeModal = true;
				});
			} else {
				this.showingNewNodeModal = true;
			}
		},
		onNewNodeModalClosed: function() {
			this.showingNewNodeModal = false;
			this.update();
		},
		selectNode: function(node) {
			if(this.selectedNode) {
				this.selectedNode = null;
				Vue.nextTick(() => {
					this.selectedNode = node;
				});
			} else {
				this.selectedNode = node;
			}
		},
	},
	template: `
<div class="flex-container">
	<div>
		<button type="button" class="btn btn-primary" v-on:click="showNewNodeModal">Add Node</button>
	</div>
	<div class="flex-content scroll-content">
		<table class="table table-sm table-row-select">
			<thead>
				<tr>
					<th>Name</th>
					<th>Operation</th>
					<th>Inputs</th>
					<th>Outputs</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr
					v-for="nodeInfo in sorted"
					:class="{selected: selectedNode && nodeInfo.Node.ID == selectedNode.ID}"
					v-on:click="selectNode(nodeInfo.Node)"
					>
					<td>{{ nodeInfo.Node.Name }}</td>
					<td>{{ $globals.ops[nodeInfo.Node.Op].Name }}</td>
					<td>{{ nodeInfo.Inputs }}</td>
					<td>{{ nodeInfo.Outputs }}</td>
					<td></td>
				</tr>
			</tbody>
		</table>
	</div>
	<div v-if="selectedNode" class="flex-content scroll-content">
		<pipeline-selected-pane :node="selectedNode" :nodes="nodes" :datasets="datasets" :workspaces="workspaces" v-on:update="update"></pipeline-selected-pane>
	</div>
	<add-exec-node v-if="showingNewNodeModal" v-on:closed="onNewNodeModalClosed"></add-exec-node>
</div>
	`,
};
