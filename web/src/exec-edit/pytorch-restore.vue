<template>
<div class="small-container">
	<template v-for="(name, idx) in parentNames">
		<h4>{{ name }}</h4>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Source Prefix</label>
			<div class="col-sm-8">
				<input v-model.number="restore[idx].SrcPrefix" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Prefix of data in the parent model to load.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Destination Prefix</label>
			<div class="col-sm-8">
				<input v-model.number="restore[idx].DstPrefix" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Prefix of data in this model to restore the saved parameters to.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Skip Prefixes</label>
			<div class="col-sm-8">
				<input v-model.number="restore[idx].SkipPrefixes" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Comma-separated list of prefixes that should not be restored.
				</small>
			</div>
		</div>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			restore: [],
			parentNames: [],
		};
	},
	props: ['node', 'value'],
	created: function() {
		if(this.value) {
			this.restore = this.value;
		}

		let models = [];
		if(this.node.Parents && this.node.Parents['models']) {
			models = this.node.Parents['models'];
		}
		while(this.restore.length < models.length) {
			this.restore.push({
				SrcPrefix: '',
				DstPrefix: '',
				SkipPrefixes: '',
			});
		}
		this.parents = [];
		models.forEach((parent, idx) => {
			this.parentNames.push('unknown');
			if(parent.Type == 'n') {
				utils.request(this, 'GET', '/exec-nodes/'+parent.ID, null, (node) => {
					this.$set(this.parentNames, idx, node.Name);
				});
			} else if(parent.Type == 'd') {
				utils.request(this, 'GET', '/datasets/'+parent.ID, null, (ds) => {
					this.$set(this.parentNames, idx, ds.Name);
				});
			}
		});
		this.update();
	},
	methods: {
		update: function() {
			this.$emit('input', this.restore);
		},
	},
};
</script>