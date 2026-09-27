struct Attributes {
	float2 M0;
	float2 M1;
	float4 M2;
};

struct Varyings {
	float4 Position [[position]];
	float2 M0;
	float4 M1;
};

float4 F0(float2 l0);
float4 F1(float4 l0);
float4 F2(bool front_facing, float4 l0);

float4 F0(float2 l0) {
	return float4(l0, 0.0, 1.0);
}

float4 F1(float4 l0) {
	return (l0) * (5.0000000000e-01);
}

float4 F2(bool front_facing, float4 l0) {
	return (F1(l0)) * (float4(1.0, 0.0, 0.0, 1.0));
}

vertex Varyings Vertex(
	uint vid [[vertex_id]],
	const device Attributes* attributes [[buffer(0)]]) {
	Varyings varyings = {};
	varyings.Position = F0(attributes[vid].M0);
	varyings.M0 = attributes[vid].M1;
	varyings.M1 = F1(attributes[vid].M2);
	return varyings;
}

fragment float4 Fragment(
	Varyings varyings [[stage_in]],
	bool front_facing [[front_facing]]) {
	return F2(front_facing, varyings.M1);
}
