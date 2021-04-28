<template>
<div>
	<template v-if="dataset != null">
		<div class="border-bottom mb-3">
			<h2>Dataset: {{ dataset.Name }}</h2>
		</div>
		<p><import-modal mode="add" v-bind:dataset="dataset"></import-modal></p>
		<h4>Items</h4>
		<table class="table table-sm">
			<thead>
				<tr>
					<th>Key</th>
					<th>Format</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="(item, i) in items">
					<td><router-link :to="'/ws/'+$route.params.ws+'/datasets/'+dataset.ID+'/items/'+item.Key">{{ item.Key }}</router-link></td>
					<td>{{ item.Format }}</td>
					<td>
						<button v-on:click="deleteItem(item.Key)" class="btn btn-sm btn-danger">Delete</button>
					</td>
				</tr>
			</tbody>
		</table>
	</template>
</div>
</template>

<script>
import utils from './utils.js';
import ImportModal from './import-modal.vue';
import RenderItem from './render-item.vue';

export default {
	components: {
		'import-modal': ImportModal,
		'render-item': RenderItem,
	},
	data: function() {
		return {
			datasetID: null,
			dataset: null,
			items: [],
		};
	},
	created: function() {
		this.datasetID = this.$route.params.dsid;
		utils.request(this, 'GET', '/datasets/'+this.datasetID, null, (dataset) => {
			this.dataset = dataset;

			this.$store.commit('setRouteData', {
				dataset: this.dataset,
			});
		});
		this.fetchItems();
	},
	methods: {
		fetchItems: function() {
			utils.request(this, 'GET', '/datasets/'+this.datasetID+'/items', null, (items) => {
				this.items = items;
			});
		},
		deleteItem: function(key) {
			utils.request(this, 'DELETE', '/datasets/'+this.datasetID+'/items/'+key, null, () => {
				this.fetchItems();
			});
		},
	},
};
</script>
