<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<p>
			Use the table below to define output splits.
			Each split is configured by the percentage of the input dataset that should end up in that split.
			The percentages across splits should add up to at most 100.
		</p>
		<table class="table">
			<thead>
				<tr>
					<th>Name</th>
					<th>Percentage</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="(split, i) in params.Splits">
					<td>{{ split.Name }}</td>
					<td>{{ split.Percentage }}</td>
					<td>
						<button type="button" class="btn btn-danger" v-on:click="removeSplit(i)">Remove</button>
					</td>
				</tr>
				<tr>
					<td>
						<input type="text" class="form-control" v-model="addForm.Name" />
					</td>
					<td>
						<input type="text" class="form-control" v-model.number="addForm.Percentage" />
					</td>
					<td>
						<button type="button" class="btn btn-primary" v-on:click="addSplit">Add</button>
					</td>
				</tr>
			</tbody>
		</table>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			params: null,
			addForm: null,
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			params = JSON.parse(this.node.Params);
		} catch(e) {}
		if(!('Splits' in params)) params.Splits = [];
		this.params = params;
		this.resetForm();
	},
	methods: {
		resetForm: function() {
			this.addForm = {
				Name: '',
				Percentage: null,
			};
		},
		addSplit: function() {
			this.params.Splits.push(this.addForm);
			this.resetForm();
		},
		removeSplit: function(i) {
			this.params.Splits.splice(i, 1);
		},

		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>
