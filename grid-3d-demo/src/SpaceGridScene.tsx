import React from "react";
import { Canvas } from "@react-three/fiber";
import { OrbitControls, Stars } from "@react-three/drei";

export type Coord3 = [number, number, number];

export interface ClaimedCell {
  coord: Coord3;
  value: any;
}

interface SpaceGridSceneProps {
  cells: ClaimedCell[];
  userId: string;
}

const GRID_SIZE = 100;
const SPACING = 0.2; 

function mapCoordToPosition([x, y, z]: Coord3): [number, number, number] {
  const half = GRID_SIZE / 2;
  const px = (x - half) * SPACING;
  const py = (y - half) * SPACING;
  const pz = (z - half) * SPACING;
  return [px, py, pz];
}

const CellCube: React.FC<{ cell: ClaimedCell; userId: string }> = ({ cell, userId }) => {
  const [x, y, z] = cell.coord;
  const [px, py, pz] = mapCoordToPosition([x, y, z]);

  const value = cell.value;
  let color = "#22c55e"; 
  if (typeof value === "string" && value.startsWith("held:")) {
    const holder = value.slice("held:".length);
    if (holder !== userId) {
      color = "#ef4444";
    }
  }

  return (
    <mesh position={[px, py, pz]}>
      <boxGeometry args={[SPACING * 0.8, SPACING * 0.8, SPACING * 0.8]} />
      <meshStandardMaterial
        color={color}
        emissive={color}
        transparent
        opacity={0.7}
      />
    </mesh>
  );
};

export const SpaceGridScene: React.FC<SpaceGridSceneProps> = ({ cells, userId }) => {
  return (
    <Canvas
      camera={{ position: [5, 5, 5], fov: 60 }}
      style={{ width: "100%", height: "100%" }}
    >
      <color attach="background" args={["#020617"]} />
      <Stars radius={100} depth={50} count={5000} factor={4} fade />

      <ambientLight intensity={0.4} />
      <pointLight position={[10, 10, 10]} intensity={1.2} />

      <OrbitControls enableDamping />
      <group>
        {cells.map((cell) => (
          <CellCube
            key={cell.coord.join(":")}
            cell={cell}
            userId={userId}
          />
        ))}
      </group>
    </Canvas>
  );
};
