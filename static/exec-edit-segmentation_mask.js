import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			dims: [0, 0],
		};
	},
	created: function() {
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				this.dims = s.Dims;
			} catch(e) {}
		});
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Dims: this.dims,
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/queries');
			});
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Output Width</label>
			<div class="col-sm-10">
				<input v-model.number="dims[0]" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Output Height</label>
			<div class="col-sm-10">
				<input v-model.number="dims[1]" type="text" class="form-control">
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
