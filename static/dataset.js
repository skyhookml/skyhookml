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
			viewingItem: null,
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
			this.viewingItem = this.items[i];
		},
	},
	template: `
<div>
	<template v-if="dataset != null">
		<h2>Dataset: {{ dataset.Name }}</h2>
		<template v-if="viewingItem == null">
			<p><import-modal v-bind:dataset="dataset"></import-modal>
			<h4>Items</h4>
			<table class="table">
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
		<template v-else>
			<render-item v-bind:dataType="dataset.DataType" v-bind:item="viewingItem"></render-item>
		</template>
	</template>
</div>
	`,
};
