import PytorchTrainGeneric from './exec-edit-pytorch_train-generic.js';
export default PytorchTrainGeneric({
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('Mode' in this.params)) this.$set(this.params, 'Mode', 'x');
		if(!('Width' in this.params)) this.$set(this.params, 'Width', 0);
		if(!('Height' in this.params)) this.$set(this.params, 'Height', 0);
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Mode</label>
		<div class="col-sm-8">
			<select v-model="params.Mode" class="form-select">
				<option value="s">YOLOv5s</option>
				<option value="m">YOLOv5m</option>
				<option value="l">YOLOv5l</option>
				<option value="x">YOLOv5x</option>
			</select>
			<small class="form-text text-muted">
				Choose a model size: small (s), medium (m), large (l), and x-large (x).
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Width</label>
		<div class="col-sm-8">
			<input v-model.number="params.Width" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this width (must be a multiple of 32). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Height</label>
		<div class="col-sm-8">
			<input v-model.number="params.Height" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this height (must be a multiple of 32). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Validation Percentage</label>
		<div class="col-sm-8">
			<input v-model.number="params.ValPercent" type="text" class="form-control">
			<small class="form-text text-muted">
				Use this percentage of the input data for validation. The rest will be used for training.
			</small>
		</div>
	</div>
</div>
	`,
});
