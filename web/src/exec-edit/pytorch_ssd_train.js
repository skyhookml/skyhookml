import PytorchTrainGeneric from './pytorch_train-generic.js';
export default PytorchTrainGeneric({
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('Mode' in this.params)) this.$set(this.params, 'Mode', 'mb2-ssd-lite');
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Mode</label>
		<div class="col-sm-8">
			<select v-model="params.Mode" class="form-select">
				<option value="vgg16-ssd">VGG+SSD</option>
				<option value="mb1-ssd">MobileNetv1+SSD</option>
				<option value="mb1-ssd-lite">MobileNetv1+SSD-Lite</option>
				<option value="sq-ssd-lite">SqueezeNet+SSD-Lite</option>
				<option value="mb2-ssd-lite">MobileNetv2+SSD-Lite</option>
				<option value="mb3-large-ssd-lite">MobileNetv3-Large+SSD-Lite</option>
				<option value="mb3-small-ssd-lite">MobileNetv3-Small+SSD-Lite</option>
			</select>
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
