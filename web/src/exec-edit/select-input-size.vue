<!--
Component for selecting model input resolution.
Offers three resolution modes which should be stored in some parameter object.
-->

<template>
<div class="small-container">
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Resize Mode</label>
		<div class="col-sm-8">
			<div class="form-check">
				<input class="form-check-input" type="radio" v-model="params.Mode" value="scale-down" @change="update">
				<label class="form-check-label">Scale-Down: Downsize image so that its maximum dimension is at most a configurable value.</label>
			</div>
			<div class="form-check">
				<input class="form-check-input" type="radio" v-model="params.Mode" value="fixed" @change="update">
				<label class="form-check-label">Fixed: Resize to a fixed resolution.</label>
			</div>
			<div class="form-check">
				<input class="form-check-input" type="radio" v-model="params.Mode" value="keep" @change="update">
				<label class="form-check-label">Keep: Use input images as-is without resizing.</label>
			</div>
		</div>
	</div>
	<template v-if="params.Mode == 'scale-down'">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Max Dimension</label>
			<div class="col-sm-8">
				<input v-model.number="params.MaxDimension" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Scale down images so that their maximum dimension is at most this value.
					For example, if set to 640, a 1280x720 input image would be scaled to 640x360.
				</small>
			</div>
		</div>
	</template>
	<template v-if="params.Mode == 'fixed'">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Width</label>
			<div class="col-sm-8">
				<input v-model.number="params.Width" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Resize the image to this width.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Height</label>
			<div class="col-sm-8">
				<input v-model.number="params.Height" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Resize the image to this height.
				</small>
			</div>
		</div>
	</template>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Multiple</label>
		<div class="col-sm-8">
			<input v-model.number="params.Multiple" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Ensure the final image resolution is a multiple of this number on both dimensions, by rounding down.
			</small>
		</div>
	</div>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			params: {},
		};
	},
	props: ['value'],
	created: function() {
		let params = {};
		if(this.value) {
			for(let [k, v] of Object.entries(this.value)) {
				params[k] = v;
			}
		}
		if(!('Mode' in params)) params['Mode'] = 'keep';
		if(!('MaxDimension' in params)) params['MaxDimension'] = 640;
		if(!('Width' in params)) params['Width'] = 256;
		if(!('Height' in params)) params['Height'] = 256;
		if(!('Multiple' in params)) params['Multiple'] = 1;
		this.params = params;
	},
	methods: {
		update: function() {
			this.$emit('input', this.params);
			this.$emit('change');
		},
	},
};
</script>
