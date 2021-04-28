<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Materialize</label>
			<div class="col-sm-8">
				<div class="form-check">
					<input class="form-check-input" type="checkbox" v-model="params.Materialize" @change="update">
					<label class="form-check-label">
						Materialize the images in the output dataset (fetch the images immediately).
					</label>
				</div>
				<small class="form-text text-muted">If unchecked, the images will be loaded lazily upon access.</small>
			</div>
		</div>
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
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			params = JSON.parse(this.node.Params);
		} catch(e) {}
		if(!('Materialize' in params)) params.Materialize = false;
		this.params = params;
	},
	methods: {
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