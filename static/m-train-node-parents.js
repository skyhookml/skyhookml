Vue.component('m-train-node-parents', {
	data: function() {
		return {
			selected: '',
		};
	},
	props: [
		'node', 'nodes', 'label',
	],
	methods: {
		add: function() {
			let nodeID = parseInt(this.selected);
			this.$emit('add', nodeID);
		},
	},
	watch: {
		node: function() {
			this.selected = '';
		},
	},
	template: `
<table class="table table-sm">
	<thead>
		<tr><th colspan="2">{{ label }}</th></tr>
	</thead>
	<tbody>
		<tr v-for="(parentID, i) in node.ParentIDs" :key="i">
			<td>{{ nodes[parentID].Name }}</td>
			<td><button type="button" class="btn btn-danger btn-sm" v-on:click="$emit('remove', i)">Remove</button></td>
		</tr>
		<tr>
			<td>
				<select v-model="selected" class="form-control">
					<template v-for="node in nodes">
						<option :value="node.ID">{{ node.Name }}</option>
					</template>
				</select>
			</td>
			<td><button type="button" class="btn btn-success btn-sm" v-on:click="add">Add</button></td>
		</tr>
	</tbody>
</table>
	`,
});
