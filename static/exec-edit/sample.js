import utils from '../utils.js';

export default {
	data: function() {
		return {
			params: null,
			addKeyInput: '',
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			params = JSON.parse(this.node.Params);
		} catch(e) {}
		if(!('Mode' in params)) params.Mode = 'count';
		if(!('Count' in params)) params.Count = 4;
		if(!('Percentage' in params)) params.Percentage = 10;
		if(!('Keys' in params)) params.Keys = [];
		this.params = params;
	},
	methods: {
		addKey: function() {
			this.params.Keys.push(this.addKeyInput);
			this.addKeyInput = '';
		},
		removeKey: function(i) {
			this.params.Keys.splice(i, 1);
		},

		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Mode</label>
			<div class="col-sm-8">
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.Mode" value="count">
					<label class="form-check-label">Count: Sample a given number of items.</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.Mode" value="percentage">
					<label class="form-check-label">Percentage: Sample a given percentage of items in the input dataset.</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.Mode" value="direct">
					<label class="form-check-label">Direct: Sample items matching a specified list of keys.</label>
				</div>
			</div>
		</div>
		<template v-if="params.Mode == 'count'">
			<div class="form-group row">
				<label class="col-sm-4 col-form-label">Count</label>
				<div class="col-sm-8">
					<input v-model.number="params.Count" type="text" class="form-control">
					<small class="form-text text-muted">
						The number of items to sample from the input dataset(s).
					</small>
				</div>
			</div>
		</template>
		<template v-if="params.Mode == 'percentage'">
			<div class="form-group row">
				<label class="col-sm-4 col-form-label">Percentage</label>
				<div class="col-sm-8">
					<div class="input-group">
						<input v-model.number="params.Percentage" type="text" class="form-control">
						<span class="input-group-text">%</span>
					</div>
					<small class="form-text text-muted">
						The percentage of items to sample from the input dataset(s).
					</small>
				</div>
			</div>
		</template>
		<template v-if="params.Mode == 'direct'">
			<div class="form-group row">
				<label class="col-sm-4 col-form-label">Keys</label>
				<div class="col-sm-8">
					<table class="table">
						<thead>
							<tr>
								<th>Key</th>
								<th></th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="(key, i) in params.Keys" :key="key">
								<td>{{ key }}</td>
								<td>
									<button type="button" class="btn btn-danger" v-on:click="removeKey(i)">Remove</button>
								</td>
							</tr>
							<tr>
								<td>
									<input type="text" class="form-control" v-model="addKeyInput" />
								</td>
								<td>
									<button type="button" class="btn btn-primary" v-on:click="addKey">Add</button>
								</td>
							</tr>
						</tbody>
					</table>
					<small class="form-text text-muted">
						The specific keys to sample.
					</small>
				</div>
			</div>
		</template>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
