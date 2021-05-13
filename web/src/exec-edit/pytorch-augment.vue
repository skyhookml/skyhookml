<template>
<div class="small-container">
	<template v-for="(d, i) in augment">
		<template v-if="d.Op == 'random_resize'">
			<h3>Random Resize <button type="button" class="btn btn-sm btn-danger" v-on:click="removeAugmentation(i)">Remove</button></h3>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Minimum Width</label>
				<div class="col-sm-10">
					<input v-model.number="d.P.MinWidth" type="text" class="form-control" @change="update">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Minimum Height</label>
				<div class="col-sm-10">
					<input v-model.number="d.P.MinHeight" type="text" class="form-control" @change="update">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Maximum Width</label>
				<div class="col-sm-10">
					<input v-model.number="d.P.MaxWidth" type="text" class="form-control" @change="update">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Maximum Height</label>
				<div class="col-sm-10">
					<input v-model.number="d.P.MaxHeight" type="text" class="form-control" @change="update">
				</div>
			</div>
			<div class="form-group row">
				<div class="col-sm-2">Options</div>
				<div class="col-sm-10">
					<div class="form-check">
						<input class="form-check-input" type="checkbox" v-model="d.P.KeepRatio" @change="update">
						<label class="form-check-label">
							Keep aspect ratio the same when resizing. In this case, only min/max width are used.
						</label>
					</div>
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Multiple</label>
				<div class="col-sm-10">
					<input v-model.number="d.P.Multiple" type="text" class="form-control" @change="update">
					<small class="form-text text-muted">
						When resizing, round random width/height up to the next higher multiple of this number.
					</small>
				</div>
			</div>
		</template>
		<template v-else-if="d.Op == 'crop'">
			<h3>Cropping <button type="button" class="btn btn-sm btn-danger" v-on:click="removeAugmentation(i)">Remove</button></h3>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">X Factor</label>
				<div class="col-sm-10">
					<input v-model="d.P.Width" type="text" class="form-control" @change="update">
					<small class="form-text text-muted">
						Crop width is image width multiplied by this factor. Express either as a fraction (e.g., 768/1024) or a decimal (e.g., 0.75).
					</small>
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Y Factor</label>
				<div class="col-sm-10">
					<input v-model="d.P.Height" type="text" class="form-control" @change="update">
					<small class="form-text text-muted">
						Crop height is image height multiplied by this factor. Express either as a fraction (e.g., 768/1024) or a decimal (e.g., 0.75).
					</small>
				</div>
			</div>
		</template>
		<template v-else-if="d.Op == 'flip'">
			<h3>Flipping <button type="button" class="btn btn-sm btn-danger" v-on:click="removeAugmentation(i)">Remove</button></h3>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Mode</label>
				<div class="col-sm-10">
					<select v-model="d.P.Mode" class="form-select" @change="update">
						<option value="both">Both</option>
						<option value="horizontal">Horizontal Flip Only</option>
						<option value="vertical">Vertical Flip Only</option>
					</select>
				</div>
			</div>
		</template>
		<template v-else>
			<h3>Unknown Augmentation: {{ d.Op }} <button type="button" class="btn btn-sm btn-danger" v-on:click="removeAugmentation(i)">Remove</button></h3>
		</template>
	</template>
	<h3>Add Data Augmentation</h3>
	<fieldset class="form-group">
		<div class="row">
			<legend class="col-form-label col-sm-2 pt-0">Type</legend>
			<div class="col-sm-10">
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="addForm.op" value="random_resize">
					<label class="form-check-label">Random Resize</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="addForm.op" value="crop">
					<label class="form-check-label">Cropping</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="addForm.op" value="flip">
					<label class="form-check-label">Horizontal and Vertical Flipping</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="addForm.op" value="rotate">
					<label class="form-check-label">Random Rotation</label>
				</div>
			</div>
		</div>
	</fieldset>
	<div class="form-group row">
		<div class="col-sm-10">
			<button type="button" class="btn btn-primary" v-on:click="addAugmentation">Add Augmentation</button>
		</div>
	</div>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			augment: [],
			addForm: {},
		};
	},
	props: ['node', 'value'],
	created: function() {
		this.augment = this.value.map((el) => {
			return {
				Op: el.Op,
				P: JSON.parse(el.Params),
			};
		});
		this.resetAddForm();
	},
	methods: {
		resetAddForm: function() {
			this.addForm = {
				op: '',
			};
		},
		addAugmentation: function() {
			let op = this.addForm.op;
			if(!op) {
				return;
			}
			this.resetAddForm();
			let p = {};
			if(op == 'random_resize') {
				p.MinWidth = 0;
				p.MinHeight = 0;
				p.MaxWidth = 0;
				p.MaxHeight = 0;
				p.KeepRatio = false;
				p.Multiple = 0;
			} else if(op == 'crop') {
				p.Width = '';
				p.Height = '';
			} else if(op == 'flip') {
				p.Mode = 'both';
			}
			this.augment.push({
				Op: op,
				P: p,
			});
			this.update();
		},
		removeAugmentation: function(i) {
			this.augment.splice(i, 1);
			this.update();
		},
		update: function() {
			let s = this.augment;
			s = s.map((el) => {
				return {
					Op: el.Op,
					Params: JSON.stringify(el.P),
				};
			});
			this.$emit('input', s);
		},
	},
};
</script>
