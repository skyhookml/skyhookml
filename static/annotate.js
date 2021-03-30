import utils from './utils.js';

export default {
	data: function() {
		return {
			annosets: [],
		};
	},
	created: function() {
		this.fetch();
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
	filters: {
		// Format the Inputs of an annotation dataset.
		niceInputs: function(inputs) {
			let datasetNames = inputs.map((input) => input.Name);
			return datasetNames.join(', ');
		},
	},
	template: `
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
				<td>{{ set.Inputs | niceInputs }}</td>
				<td>{{ $globals.dataTypes[set.Dataset.DataType] }}</td>
				<td>
					<router-link :to="'/ws/'+$route.params.ws+'/annotate/'+set.Tool+'/'+set.ID" class="btn btn-sm btn-primary">Annotate</router-link>
					<button v-on:click="removeAnnoset(set)" class="btn btn-sm btn-danger">Remove</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>
	`,
};
