import utils from './utils.js';
import ImportModal from './import-modal.js';
import RenderItem from './render-item.js';

export default {
	components: {
		'import-modal': ImportModal,
		'render-item': RenderItem,
	},
	data: function() {
		return {
			dataset: null,
			items: [],
		};
	},
	created: function() {
		const dsID = this.$route.params.dsid;
		utils.request(this, 'GET', '/datasets/'+dsID, null, (dataset) => {
			this.dataset = dataset;
		});
		utils.request(this, 'GET', '/datasets/'+dsID+'/items', null, (items) => {
			this.items = items;
		});
	},
	methods: {
		viewItem: function(i) {
			this.$router.push('/ws/'+this.$route.params.ws+'/datasets/'+this.dataset.ID+'/items/'+this.items[i].Key);
		},
	},
	template: `
<div>
	<template v-if="dataset != null">
		<div class="border-bottom mb-3">
			<h2>Dataset: {{ dataset.Name }}</h2>
		</div>
		<p><import-modal v-bind:dataset="dataset"></import-modal>
		<h4>Items</h4>
		<table class="table table-sm">
			<thead>
				<tr>
					<th>Key</th>
					<th>Format</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="(item, i) in items">
					<td><a href="#" v-on:click.prevent="viewItem(i)">{{ item.Key }}</a></td>
					<td>{{ item.Format }}</td>
				</tr>
			</tbody>
		</table>
	</template>
</div>
	`,
};
