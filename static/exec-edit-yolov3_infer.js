import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			params: {
				inputWidth: '',
				inputHeight: '',
			},
		};
	},
	created: function() {
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				if(s.InputSize) {
					this.params.inputWidth = s.InputSize[0];
					this.params.inputHeight = s.InputSize[1];
				}
			} catch(e) {}
		});
	},
	methods: {
		save: function() {
			let params = {
				InputSize: [parseInt(this.params.inputWidth), parseInt(this.params.inputHeight)],
			};
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/queries');
			});
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Input Width</label>
			<div class="col-sm-10">
				<input v-model="params.inputWidth" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Input Height</label>
			<div class="col-sm-10">
				<input v-model="params.inputHeight" type="text" class="form-control">
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
