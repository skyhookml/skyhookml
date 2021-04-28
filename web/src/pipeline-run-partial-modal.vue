<template>
<div class="modal" tabindex="-1" role="dialog" ref="modal">
	<div class="modal-dialog modal-lg" role="document">
		<div class="modal-content">
			<div class="modal-body">
				<form v-on:submit.prevent="execute">
					<div class="row mb-2">
						<label class="col-sm-4 col-form-label">Mode</label>
						<div class="col-sm-8">
							<div class="form-check">
								<input class="form-check-input" type="radio" v-model="mode" value="random">
								<label class="form-check-label">Random: Compute a fixed number of random outputs.</label>
							</div>
							<div class="form-check">
								<input class="form-check-input" type="radio" v-model="mode" value="dataset">
								<label class="form-check-label">Dataset: Compute only outputs with keys matching those in another dataset.</label>
							</div>
							<div class="form-check">
								<input class="form-check-input" type="radio" v-model="mode" value="direct">
								<label class="form-check-label">Direct: Compute outputs matching a specified list of keys.</label>
							</div>
						</div>
					</div>
					<template v-if="mode == 'random'">
						<div class="form-group row">
							<label class="col-sm-4 col-form-label">Count</label>
							<div class="col-sm-8">
								<input v-model.number="count" type="text" class="form-control">
								<small class="form-text text-muted">
									The number of output items to compute.
								</small>
							</div>
						</div>
					</template>
					<template v-if="mode == 'dataset'">
						<div class="form-group row">
							<label class="col-sm-4 col-form-label">Dataset</label>
							<div class="col-sm-8">
								<select v-model="optionIdx" class="form-select" required>
									<template v-for="(opt, idx) in options">
										<option :value="idx">{{ opt.Label }}</option>
									</template>
								</select>
							</div>
						</div>
					</template>
					<template v-if="mode == 'direct'">
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
										<tr v-for="(key, i) in keys" :key="key">
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
					<div class="form-group row">
						<div class="col-sm-8">
							<button type="submit" class="btn btn-primary">Run Node Partially</button>
						</div>
					</div>
				</form>
			</div>
		</div>
	</div>
</div>
</template>

<script>
import utils from './utils.js';
import get_parent_options from './get-parent-options.js';

export default {
	data: function() {
		return {
			// Partial execution mode, one of 'random', 'dataset', or 'direct'.
			mode: 'random',
			// If mode=='random', the number of output items to compute.
			count: 4,
			// If mode=='dataset', the dataset specifying the keys to compute.
			// this.options[optionIdx] is an ExecParent object.
			optionIdx: null,
			// If mode=='direct', the list of keys to compute.
			keys: [],

			// List of parent options for mode=='dataset' selection.
			options: [],
			// If mode=='direct', this provides input value for adding new key to this.keys.
			addKeyInput: '',
		};
	},
	props: [
		// The selected node that we want to run.
		'node',
	],
	created: function() {
		// Populate this.options.
		get_parent_options(this.$route.params.ws, this, (options) => {
			this.options = options;
		});
	},
	mounted: function() {
		$(this.$refs.modal).modal('show');
	},
	methods: {
		execute: function() {
			let params;
			if(this.mode == 'dataset') {
				params = {
					Mode: 'dataset',
					ParentSpec: this.options[this.optionIdx],
				};
			} else {
				params = {
					Mode: this.mode,
					Count: this.count,
					Keys: this.keys,
				};
			}

			(async () => {
				let job = null;
				try {
					job = await utils.request(this, 'POST', '/exec-nodes/'+this.node.ID+'/incremental', JSON.stringify(params));
				} catch(e) {
					console.log('error running incrementally', e);
					this.$globals.app.setError(e.responseText);
				}
				$(this.$refs.modal).modal('hide');
				this.$emit('closed');
				if(!job) {
					return;
				}
				this.$router.push('/ws/'+this.$route.params.ws+'/jobs/'+job.ID);
			})();
		},

		addKey: function() {
			this.keys.push(this.addKeyInput);
			this.addKeyInput = '';
		},
		removeKey: function(i) {
			this.keys.splice(i, 1);
		},
	},
};
</script>