<template>
<div class="small-container">
	<h3>Add Annotation Dataset</h3>
	<template v-if="!addForm.tool">
		<p>Select an annotation tool:</p>
		<template v-for="tool in tools">
			<div class="card my-2" style="max-width: 800px" role="button" v-on:click="selectTool(tool.ID)">
				<div class="card-body">
					<h5 class="card-title">{{ tool.Name }}</h5>
					<p class="card-text">{{ tool.Help }}</p>
				</div>
			</div>
		</template>
	</template>
	<template v-else>
		<form v-on:submit.prevent="addAnnoset">
			<div class="row mb-2">
				<label class="col-sm-4 col-form-label">Create Dataset?</label>
				<div class="col-sm-8">
					<div class="form-check">
						<input class="form-check-input" type="radio" v-model="addForm.datasetMode" value="new">
						<label class="form-check-label">Create a New Dataset</label>
					</div>
					<div class="form-check">
						<input class="form-check-input" type="radio" v-model="addForm.datasetMode" value="existing">
						<label class="form-check-label">Add Annotations to an Existing Dataset</label>
					</div>
				</div>
			</div>
			<template v-if="addForm.datasetMode == 'new'">
				<div class="row mb-2">
					<label class="col-sm-4 col-form-label">Name</label>
					<div class="col-sm-8">
						<input v-model="addForm.datasetName" type="text" class="form-control" required>
						<small class="form-text text-muted">A name for these annotations.</small>
					</div>
				</div>
				<div class="row mb-2" v-if="addForm.toolObj">
					<label class="col-sm-4 col-form-label">Data Type</label>
					<div class="col-sm-8">
						<template v-if="!addForm.toolObj.DataType">
							<div class="form-check" v-for="(label, dt) in addForm.toolObj.DataTypes">
								<input class="form-check-input" type="radio" v-model="addForm.datasetType" name="datasetType" :value="dt" required>
								<label class="form-check-label">{{ label }}</label>
							</div>
						</template>
						<template v-else>
							<input type="text" readonly class="form-control-plaintext" :value="addForm.datasetType" />
						</template>
					</div>
				</div>
			</template>
			<template v-if="addForm.datasetMode == 'existing'">
				<div class="row mb-2" v-if="addForm.toolObj">
					<label class="col-sm-4 col-form-label">Existing Dataset</label>
					<div class="col-sm-8">
						<select v-model="addForm.datasetID" class="form-select" required>
							<template v-for="ds in datasets">
								<option
									v-if="ds.Type == 'data' && (!addForm.toolObj.DataTypes || addForm.toolObj.DataTypes[ds.DataType]) && (!addForm.toolObj.DataType || addForm.toolObj.DataType == ds.DataType)"
									:key="ds.ID"
									:value="ds.ID">
									{{ ds.Name }}
								</option>
							</template>
						</select>
						<small class="form-text text-muted">An existing dataset to extend with new annotations.</small>
					</div>
				</div>
			</template>
			<template v-for="(input, i) in addForm.toolObj.Inputs">
				<div class="row mb-2">
					<label class="col-sm-4 col-form-label">Input {{ input.Name }}</label>
					<div class="col-sm-8">
						<select v-model="addForm.inputIDs[i]" class="form-select" required>
							<template v-for="ds in datasets">
								<!-- Only show datasets that match the type of this input. -->
								<option
									v-if="!input.DataTypes || input.DataTypes.includes(ds.DataType)"
									:key="ds.ID"
									:value="ds.ID">
									{{ ds.Name }}
								</option>
							</template>
						</select>
						<small v-if="input.Help" class="form-text text-muted">{{ input.Help }}</small>
					</div>
				</div>
			</template>
			<div class="row mb-2">
				<div class="col-sm-12">
					<button type="submit" class="btn btn-primary">Add Annotation Dataset</button>
				</div>
			</div>
		</form>
	</template>
</div>
</template>

<script>
import utils from './utils.js';

export default {
	data: function() {
		return {
			datasets: [],
			addForm: {
				// the selected tool ID and actual object
				// object is filled in by changedTool
				tool: null,
				toolObj: null,

				// either 'existing' or 'new', whether to create a new dataset
				datasetMode: 'new',

				// name and type of dataset in case datasetMode=='new'
				// if toolObj.DataType is set (only one option), then datasetType is not configurable
				datasetName: '',
				datasetType: '',

				// existing dataset ID to add annotations to
				datasetID: '',

				// input dataset IDs to use for annotation
				// corresponds to toolObj.Inputs
				inputIDs: [],
			},

			tools: {
				"shape": {
					ID: "shape",
					Name: "Object Detection and Segmentation",
					Help: "Annotate bounding boxes, polygons, lines, and other shapes. Each shape can be further labeled with a category and a track ID.",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
						Help: "Select an image or video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}],
					DataTypes: {
						'detection': 'Object Detections. Choose this type if you are annotating bounding boxes for object detection.',
						'shape': 'Shapes. Choose this type if you are annotating shapes other than bounding boxes, e.g., polygons for image segmentation.',
					},
				},
				"int": {
					ID: "int",
					Name: "Image Classification or Regression",
					Help: "Annotate integers, such as category IDs for image classification or arbitrary numbers for regression.",
					Inputs: [{
						Name: "Image/Video",
						DataTypes: ["image", "video"],
						Help: "Select an image or video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}],
					DataType: "int",
				},
				"detection-to-track": {
					ID: "detection-to-track",
					Name: "Group Detections into Tracks",
					Help: "Given video, along with object detections computed or previously annotated in the video, group together detections into tracks to derive training data for an object tracking model.",
					Inputs: [{
						Name: "Video",
						DataTypes: ["video"],
						Help: "Select a video dataset to label. If you have not imported the data yet, first head to Dashboard and then Quickstart Import.",
					}, {
						Name: "Detections",
						DataTypes: ["detection"],
						Help: "Select a dataset of object detections corresponding to the video.",
					}],
					DataType: "detection",
				},
				"geojson": {
					ID: "geojson",
					Name: "GeoJSON",
					Help: "Annotate GeoJSON objects in aerial or satellite imagery, including points, polylines, and polygons.",
					Inputs: [],
					DataType: "geojson",
				},
			},
		};
	},
	created: function() {
		utils.request(this, 'GET', '/datasets', null, (data) => {
			this.datasets = data;
		});
	},
	methods: {
		selectTool: function(tool) {
			// update the cached addForm.toolObj
			// if tool has a single DataType, we set datasetType
			this.addForm.tool = tool;
			let toolObj = this.tools[this.addForm.tool];
			this.addForm.toolObj = toolObj;
			this.addForm.inputIDs = [];
			for(let i = 0; i < toolObj.Inputs.length; i++) {
				this.addForm.inputIDs.push(null);
			}
			this.addForm.datasetType = '';
			if(toolObj.DataType) {
				this.addForm.datasetType = toolObj.DataType;
			}
		},
		addAnnoset: function() {
			let handle = async () => {
				let datasetID;
				if(this.addForm.datasetMode == 'new') {
					let dataset;
					try {
						let params = {
							name: this.addForm.datasetName,
							data_type: this.addForm.datasetType,
						};
						dataset = await utils.request(this, 'POST', '/datasets', params);
					} catch(e) {
						return;
					}
					datasetID = dataset.ID;
				} else {
					datasetID = this.addForm.datasetID;
				}

				let annoset;
				try {
					let params = {
						ds_id: datasetID,
						inputs: this.addForm.inputIDs.join(','),
						tool: this.addForm.tool,
						params: '',
					};
					annoset = await utils.request(this, 'POST', '/annotate-datasets', params);
				} catch(e) {
					return;
				}
				this.$router.push('/ws/'+this.$route.params.ws+'/annotate/'+annoset.Tool+'/'+annoset.ID);
			};
			handle();
		},
	},
};
</script>