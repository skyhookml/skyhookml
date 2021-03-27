import utils from './utils.js';

export default {
	data: function() {
		return {
			datasets: [],
			nodes: [],
		};
	},
	created: function() {
		this.fetch();
	},
	methods: {
		fetch: function() {
			utils.request(this, 'GET', '/datasets', null, (datasets) => {
				this.datasets = datasets;
			});
			utils.request(this, 'GET', '/exec-nodes?ws='+this.$route.params.ws, null, (nodes) => {
				this.nodes = nodes;
			});
		},
	},
	template: `
<div class="flex-container">
	<div class="flex-content-big">
		<h3 class="my-2">Quickstart</h3>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/import'">
				<div class="card-body">
					<h5 class="card-title">Import Data</h5>
					<p class="card-text">Import data into SkyhookML, whether it's unlabeled image or video, or datasets with annotations.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/annotate'">
				<div class="card-body">
					<h5 class="card-title">Annotate</h5>
					<p class="card-text">Label images or videos with object detections, image classes, etc.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/train'">
				<div class="card-body">
					<h5 class="card-title">Train a Model</h5>
					<p class="card-text">Train a model on labeled datasets.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/apply'">
				<div class="card-body">
					<h5 class="card-title">Apply a Model</h5>
					<p class="card-text">Apply a pre-trained model or a model that you've trained on new images or videos.</p>
				</div>
			</router-link>
		</div>
	</div>
	<div class="flex-content scroll-content my-2">
		<h3>Datasets</h3>
		<router-link class="btn btn-primary" :to="'/ws/'+$route.params.ws+'/datasets'">Manage</router-link>
		<table class="table table-sm align-middle">
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>Data Type</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="ds in datasets">
					<td>{{ ds.Name }}</td>
					<td>{{ ds.Type }}</td>
					<td>{{ ds.DataType }}</td>
					<td>
						<router-link class="btn btn-sm btn-primary" :to="'/ws/'+$route.params.ws+'/datasets/'+ds.ID">View</router-link>
					</td>
				</tr>
			</tbody>
		</table>
	</div>
	<div class="flex-content scroll-content my-2">
		<h3>Nodes</h3>
		<router-link class="btn btn-primary" :to="'/ws/'+$route.params.ws+'/queries'">Manage</router-link>
		<table class="table table-sm align-middle">
			<thead>
				<tr>
					<th>Name</th>
					<th>Operation</th>
					<th>Data Type</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="ds in datasets">
					<td>{{ ds.Name }}</td>
					<td>{{ ds.Type }}</td>
					<td>{{ ds.DataType }}</td>
				</tr>
			</tbody>
		</table>
	</div>
</div>
	`,
};
