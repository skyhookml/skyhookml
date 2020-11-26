Vue.component('dataset', {
	data: function() {
		return {
			items: [],
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
	},
	template: `
<div>
	<h2>
		<a href="#" v-on:click.prevent="$emit('back')">Datasets</a>
		/
		{{ dataset.Name }}
	</h2>
	<p><import-modal v-bind:dataset="dataset"></import-modal>
	<h4>Items</h4>
	<table class="table">
		<thead>
			<tr>
				<th>Key</th>
				<th>Format</th>
				<th>Length</th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="item in items">
				<td>{{ item.Key }}</td>
				<td>{{ item.Format }}</td>
				<td>{{ item.Length }}</td>
			</tr>
		</tbody>
	</table>
</div>
	`,
});
