<template>
<div>
	<div class="border-bottom mb-3">
		<h2>Annotate</h2>
	</div>
	<router-link class="btn btn-primary mb-2" :to="'/ws/'+$route.params.ws+'/annotate-add'">Add Annotation Dataset</router-link>
	<table class="table table-sm align-middle">
		<thead>
			<tr>
				<th>Name</th>
				<th>Tool</th>
				<th>Inputs</th>
				<th>Data Type</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="set in annosets">
				<td>{{ set.Dataset.Name }}</td>
				<td>{{ set.Tool }}</td>
				<td>{{ niceInputs[set.ID] }}</td>
				<td>{{ $globals.dataTypes[set.Dataset.DataType] }}</td>
				<td>
					<router-link :to="'/ws/'+$route.params.ws+'/annotate/'+set.Tool+'/'+set.ID" class="btn btn-sm btn-primary">Annotate</router-link>
					<button v-on:click="removeAnnoset(set)" class="btn btn-sm btn-danger">Remove</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>
</template>

<script>
import utils from './utils.js';

export default {
	data: function() {
		return {
			annosets: [],

			// For getting labels of inputs.
			datasets: {},
			nodes: {},
		};
	},
	created: function() {
		this.fetch();
		utils.request(this, 'GET', '/datasets', null, (dsList) => {
			let datasets = {};
			for(let ds of dsList) {
				datasets[ds.ID] = ds;
			}
			this.datasets = datasets;
		});
		utils.request(this, 'GET', '/exec-nodes', null, (nodeList) => {
			let nodes = {};
			for(let node of nodeList) {
				nodes[node.ID] = node;
			}
			this.nodes = nodes;
		});
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/annotate-datasets', null, (data) => {
				this.annosets = data;
			});
		},
		removeAnnoset: function(annoset) {
			utils.request(this, 'DELETE', '/annotate-datasets/'+annoset.ID, null, () => {
				this.fetch();
			});
		},
	},
	computed: {
		// Format the Inputs of annotation datasets.
		niceInputs: function() {
			let setToNice = {};
			for(let set of this.annosets) {
				let names = [];
				for(let input of set.Inputs) {
					if(input.Type == 'd' && this.datasets[input.ID]) {
						names.push(this.datasets[input.ID].Name);
					} else if(input.Type == 'n' && this.nodes[input.ID]) {
						names.push(this.nodes[input.ID].Name + ' [' + input.Name + ']');
					} else {
						names.push('Unknown');
					}
				}
				setToNice[set.ID] = names.join(', ');
			}
			return setToNice;
		},
	},
};
</script>
