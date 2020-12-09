Vue.component('dataset', {
	data: function() {
		return {
			items: [],
			viewingItem: null,
		};
	},
	props: ['dataset'],
	created: function() {
		this.fetch();
	},
	methods: {
		fetch: function() {
			myCall('GET', '/datasets/'+this.dataset.ID+'/items', null, (items) => {
				this.items = items;
			});
		},
		viewItem: function(i) {
			this.viewingItem = this.items[i];
		},
	},
	template: `
<div>
	<h2>
		<a href="#" v-on:click.prevent="$emit('back')">Datasets</a>
		/
		{{ dataset.Name }}
	</h2>
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
		<template v-if="dataset.DataType == 'video'">
			<video controls>
				<source :src="'/items/'+viewingItem.ID+'/get?format=mp4'" type="video/mp4" />
			</video>
		</template>
		<template v-else-if="dataset.DataType == 'image'">
			<img :src="'/items/'+viewingItem.ID+'/get?format=jpeg'" />
		</template>
	</template>
</div>
	`,
});
