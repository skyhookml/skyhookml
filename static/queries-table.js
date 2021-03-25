import utils from './utils.js';
import ExecNodeParents from './exec-node-parents.js';
import AddExecNode from './add-exec-node.js';

export default {
	components: {
		'exec-node-parents': ExecNodeParents,
		'add-exec-node': AddExecNode,
	},
	data: function() {
		return {
			datasets: {},
			nodes: {},
			sorted: [],
			selectedNode: null,
			showingNewNodeModal: false,
		};
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
					node.Parents.forEach((plist) => {
						plist.forEach((parent) => {
							if(parent.Type != 'n') {
								return;
							}
							if(!needed[parent.ID]) {
								return;
							}
							ready = false;
						});
					});
					if(!ready) {
						continue;
					}
					delete needed[nodeID];

					// create a string summary of the node inputs
					let inputs = [];
					node.Parents.forEach((plist, inputIdx) => {
						let curInput = [];
						plist.forEach((parent) => {
							if(parent.Type == 'd') {
								let dataset = this.datasets[parent.ID];
								if(!dataset) {
									curInput.push('unknown dataset');
									return;
								}
								curInput.push(dataset.Name);
							} else if(parent.Type == 'n') {
								let n = this.nodes[parent.ID];
								if(!n) {
									curInput.push('unknown node');
									return;
								}
								curInput.push(n.Name+'['+parent.Name+']');
							}
						});

						let inputName;
						if(inputIdx >= node.Inputs.length) {
							inputName = 'unknown';
						} else {
							inputName = node.Inputs[inputIdx].Name;
						}

						if(curInput.length > 1) {
							let summary = curInput.join(', ');
							inputs.push(inputName+'='+'('+summary+')');
						} else if(curInput.length == 1) {
							inputs.push(inputName+'='+curInput[0]);
						} else {
							inputs.push(inputName+'=()')
						}
					});
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
		editNode: function() {
			this.$router.push('/ws/'+this.$route.params.ws+'/exec/'+this.selectedNode.Op+'/'+this.selectedNode.ID);
		},
		runNode: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.selectedNode.ID+'/run', null, (job) => {
				this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
			});
		},
		viewInteractive: function() {
			this.$router.push('/ws/'+this.$route.params.ws+'/interactive/'+this.selectedNode.ID);
		},
		deleteNode: function() {
			utils.request(this, 'DELETE', '/exec-nodes/'+this.selectedNode.ID, null, () => {
				this.update();
			});
		},

		updateParents: function() {
			let params = JSON.stringify({
				Parents: this.selectedNode.Parents,
			});
			utils.request(this, 'POST', '/exec-nodes/' + this.selectedNode.ID, params, () => {
				this.update();
			});
		},
		addParent: function(inputIdx, parent) {
			this.selectedNode.Parents[inputIdx].push(parent);
			this.updateParents();
		},
		removeParent: function(inputIdx, idx) {
			this.selectedNode.Parents[inputIdx] = this.selectedNode.Parents[inputIdx].filter((parent, i) => i != idx);
			this.updateParents();
		},
		setParent: function(inputIdx, parent) {
			if(parent) {
				this.selectedNode.Parents[inputIdx] = [parent];
			} else {
				this.selectedNode.Parents[inputIdx] = [];
			}
			this.updateParents();
		},
	},
	template: `
<div class="flex-container">
	<div>
		<button type="button" class="btn btn-primary" v-on:click="showNewNodeModal">Add Node</button>
	</div>
	<div class="flex-content scroll-content">
		<table class="table table-row-select">
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
					<td>{{ nodeInfo.Node.Op }}</td>
					<td>{{ nodeInfo.Inputs }}</td>
					<td>{{ nodeInfo.Outputs }}</td>
					<td></td>
				</tr>
			</tbody>
		</table>
	</div>
	<div v-if="selectedNode" class="flex-content scroll-content">
		<hr />
		<h4 class="my-2">{{ selectedNode.Name }} ({{ selectedNode.Op }})</h4>
		<h5 class="my-2">Inputs</h5>
		<div v-for="(plist, inputIdx) in selectedNode.Parents" class="my-2">
			<exec-node-parents
				:node="selectedNode"
				:inputIdx="inputIdx"
				:nodes="nodes"
				:datasets="datasets"
				v-on:add="addParent(inputIdx, $event)"
				v-on:remove="removeParent(inputIdx, $event)"
				v-on:set="setParent(inputIdx, $event)"
				>
			</exec-node-parents>
		</div>
		<h5 class="my-2">Outputs</h5>
		<ul>
			<li v-for="output in selectedNode.Outputs">
				{{ output.Name }} ({{ output.DataType }})
			</li>
		</ul>
	</div>
	<add-exec-node v-if="showingNewNodeModal" v-on:closed="onNewNodeModalClosed"></add-exec-node>
</div>
	`,
};
