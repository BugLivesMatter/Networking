import { Canvas, useFrame } from '@react-three/fiber'
import { Html, Line, OrbitControls, Sparkles } from '@react-three/drei'
import { useEffect, useMemo, useRef, useState } from 'react'
import type { Group, Mesh } from 'three'
import {
  AdditiveBlending,
  BufferGeometry,
  Float32BufferAttribute,
} from 'three'
import { statusColor } from './demoCluster'
import type { ClusterService, HealthStatus } from './types'

interface ClusterSceneProps {
  services: ClusterService[]
  selectedId: string
  onSelect: (serviceId: string) => void
}

type Vec3 = [number, number, number]

const replicaOffsets: Vec3[] = [
  [-0.18, 0.08, 0.05],
  [0.18, -0.08, -0.05],
  [0.02, 0.2, -0.14],
  [-0.04, -0.22, 0.14],
  [0.24, 0.16, 0.13],
]

function ClusterCube() {
  const { edgeGeometry, vertexGeometry } = useMemo(() => {
    const vertices: Vec3[] = Array.from({ length: 8 }, (_, index) => [
      ((index >> 0) & 1) === 1 ? 2.8 : -2.8,
      ((index >> 1) & 1) === 1 ? 2.8 : -2.8,
      ((index >> 2) & 1) === 1 ? 2.8 : -2.8,
    ])
    const edgePositions: number[] = []

    for (let vertex = 0; vertex < vertices.length; vertex++) {
      for (let dimension = 0; dimension < 3; dimension++) {
        const neighbour = vertex ^ (1 << dimension)
        if (vertex >= neighbour) continue
        edgePositions.push(...vertices[vertex], ...vertices[neighbour])
      }
    }

    return {
      edgeGeometry: new BufferGeometry().setAttribute('position', new Float32BufferAttribute(edgePositions, 3)),
      vertexGeometry: new BufferGeometry().setAttribute('position', new Float32BufferAttribute(vertices.flat(), 3)),
    }
  }, [])

  useEffect(() => () => {
    edgeGeometry.dispose()
    vertexGeometry.dispose()
  }, [edgeGeometry, vertexGeometry])

  return (
    <group>
      <lineSegments geometry={edgeGeometry}>
        <lineBasicMaterial color="#9eb2cf" transparent opacity={0.34} depthWrite={false} />
      </lineSegments>
      <points geometry={vertexGeometry}>
        <pointsMaterial color="#d2e0f5" size={0.065} transparent opacity={0.62} sizeAttenuation />
      </points>
    </group>
  )
}

const connectionStatus = (source: ClusterService, target: ClusterService): HealthStatus => {
  if (source.status === 'unhealthy' || target.status === 'unhealthy') return 'unhealthy'
  if (source.status === 'degraded' || target.status === 'degraded') return 'degraded'
  if (source.status === 'starting' || target.status === 'starting') return 'starting'
  return 'healthy'
}

function Signal({ start, end, status, offset }: {
  start: Vec3
  end: Vec3
  status: HealthStatus
  offset: number
}) {
  const ref = useRef<Mesh>(null)

  useFrame(({ clock }) => {
    if (!ref.current) return
    const progress = (clock.elapsedTime * 0.18 + offset) % 1
    const bend = Math.sin(progress * Math.PI) * 0.16
    ref.current.position.set(
      start[0] + (end[0] - start[0]) * progress,
      start[1] + (end[1] - start[1]) * progress + bend,
      start[2] + (end[2] - start[2]) * progress - bend * 0.45,
    )
  })

  return (
    <mesh ref={ref}>
      <octahedronGeometry args={[0.04, 0]} />
      <meshBasicMaterial color={statusColor[status]} blending={AdditiveBlending} transparent opacity={0.9} />
    </mesh>
  )
}

function Synapse({ source, target }: { source: ClusterService; target: ClusterService }) {
  const status = connectionStatus(source, target)
  const midpoint: Vec3 = [
    (source.position[0] + target.position[0]) / 2,
    (source.position[1] + target.position[1]) / 2 + 0.18,
    (source.position[2] + target.position[2]) / 2 - 0.12,
  ]

  return (
    <group>
      <Line
        points={[source.position, midpoint, target.position]}
        color={statusColor[status]}
        lineWidth={status === 'healthy' ? 0.55 : 1.3}
        transparent
        opacity={status === 'healthy' ? 0.28 : 0.72}
      />
      <Signal start={source.position} end={target.position} status={status} offset={(source.position[0] + 4) * 0.11} />
      <Signal start={source.position} end={target.position} status={status} offset={(source.position[2] + 5) * 0.17} />
    </group>
  )
}

function WorkloadCell({ service, selected, onSelect }: {
  service: ClusterService
  selected: boolean
  onSelect: () => void
}) {
  const group = useRef<Group>(null)
  const anchor = useRef<Mesh>(null)
  const color = statusColor[service.status]

  useFrame(({ clock }, delta) => {
    if (group.current) group.current.rotation.y += delta * (selected ? 0.12 : 0.035)
    if (anchor.current) {
      const pulse = 1 + Math.sin(clock.elapsedTime * 2 + service.position[0]) * 0.08
      anchor.current.scale.setScalar(pulse)
    }
  })

  return (
    <group position={service.position}>
      <group
        ref={group}
        onClick={event => {
          event.stopPropagation()
          onSelect()
        }}
        onPointerOver={() => { document.body.style.cursor = 'pointer' }}
        onPointerOut={() => { document.body.style.cursor = 'default' }}
      >
        <mesh>
          <boxGeometry args={[selected ? 0.92 : 0.78, selected ? 0.72 : 0.62, selected ? 0.82 : 0.7]} />
          <meshBasicMaterial color={color} wireframe transparent opacity={selected ? 0.3 : 0.105} />
        </mesh>

        <mesh ref={anchor}>
          <octahedronGeometry args={[selected ? 0.105 : 0.085, 0]} />
          <meshBasicMaterial color={color} transparent opacity={0.6} />
        </mesh>

        {service.instances.map((instance, index) => {
          const offset = replicaOffsets[index % replicaOffsets.length]
          const instanceColor = statusColor[instance.status]
          return (
            <group key={instance.id} position={offset}>
              <mesh>
                <icosahedronGeometry args={[selected ? 0.16 : 0.135, 2]} />
                <meshStandardMaterial
                  color={instanceColor}
                  emissive={instanceColor}
                  emissiveIntensity={instance.status === 'healthy' ? 0.55 : 1.8}
                  roughness={0.32}
                  metalness={0.08}
                />
              </mesh>
              <Line
                points={[[0, 0, 0], [-offset[0], -offset[1], -offset[2]]]}
                color={instanceColor}
                lineWidth={0.55}
                transparent
                opacity={0.38}
              />
            </group>
          )
        })}

        {replicaOffsets.slice(service.instances.length, service.instances.length + 3).map((offset, index) => (
          <mesh key={`probe-${index}`} position={[offset[0] * 1.18, offset[1] * 1.18, offset[2] * 1.18]}>
            <sphereGeometry args={[0.025, 8, 8]} />
            <meshBasicMaterial color={color} transparent opacity={0.24} />
          </mesh>
        ))}
      </group>

      <Html center position={[0, -0.62, 0]} distanceFactor={8.5} style={{ pointerEvents: 'none' }}>
        <div className={`node-label ${selected ? 'node-label--selected' : ''}`}>
          <span>{service.name}</span>
          <small>{service.instances.length} {service.instances.length === 1 ? 'replica' : 'replicas'}</small>
        </div>
      </Html>
    </group>
  )
}

function SceneContent({ services, selectedId, onSelect }: ClusterSceneProps) {
  const root = useRef<Group>(null)
  const connections = useMemo(() => services.flatMap(source => source.dependencies.flatMap(targetId => {
    const target = services.find(service => service.id === targetId)
    return target ? [{ source, target }] : []
  })), [services])

  useFrame(({ clock }) => {
    if (!root.current) return
    root.current.position.y = Math.sin(clock.elapsedTime * 0.24) * 0.035
  })

  return (
    <>
      <color attach="background" args={["#050506"]} />
      <fog attach="fog" args={["#050506", 9, 20]} />
      <ambientLight intensity={0.72} />
      <directionalLight position={[-4, 6, 7]} intensity={1.4} color="#dce9ff" />
      <pointLight position={[4, -3, 4]} intensity={10} color="#6b87b5" distance={14} />
      <Sparkles count={44} scale={[9, 7, 7]} size={0.65} speed={0.08} opacity={0.13} color="#d5e3f8" />

      <group ref={root}>
        <ClusterCube />
        {connections.map(({ source, target }) => (
          <Synapse key={`${source.id}-${target.id}`} source={source} target={target} />
        ))}
        {services.map(service => (
          <WorkloadCell
            key={service.id}
            service={service}
            selected={service.id === selectedId}
            onSelect={() => onSelect(service.id)}
          />
        ))}
      </group>

      <OrbitControls
        makeDefault
        enablePan={false}
        enableDamping
        minDistance={7.5}
        maxDistance={15}
        autoRotate
        autoRotateSpeed={0.32}
        dampingFactor={0.06}
        minPolarAngle={Math.PI * 0.2}
        maxPolarAngle={Math.PI * 0.8}
      />
    </>
  )
}

function CompatibilityView({ services }: { services: ClusterService[] }) {
  return (
    <div className="compatibility-view" role="img" aria-label="Static cluster topology fallback">
      <svg viewBox="0 0 640 520" aria-hidden="true">
        <g className="fallback-frame">
          <path d="M120 110 420 65 545 155 242 205Z M120 110 120 360 242 455 242 205 M242 455 545 390 545 155" />
        </g>
        {services.map((service, index) => {
          const x = 230 + (service.position[0] + 3) * 45
          const y = 250 - service.position[1] * 55 + service.position[2] * 12
          return <circle key={service.id} cx={x} cy={y} r={index === 1 ? 8 : 6} className={`fallback-node fallback-node--${service.status}`} />
        })}
      </svg>
      <div><strong>Compatibility topology</strong><span>WebGL 2 is unavailable in this browser. Enable hardware acceleration for the interactive 3D view.</span></div>
    </div>
  )
}

export default function ClusterScene(props: ClusterSceneProps) {
  const [contextLost, setContextLost] = useState(false)
  const webGL2Available = useMemo(() => {
    try {
      const canvas = document.createElement('canvas')
      return Boolean(canvas.getContext('webgl2', { failIfMajorPerformanceCaveat: false }))
    } catch {
      return false
    }
  }, [])

  if (!webGL2Available || contextLost) {
    return <CompatibilityView services={props.services} />
  }

  return (
    <Canvas
      fallback={<CompatibilityView services={props.services} />}
      camera={{ position: [6.8, 4.7, 8.6], fov: 43 }}
      dpr={[1, 1.4]}
      performance={{ min: 0.55 }}
      gl={{ antialias: true, alpha: false, powerPreference: 'default' }}
      onCreated={({ gl }) => {
        gl.domElement.addEventListener('webglcontextlost', event => {
          event.preventDefault()
          setContextLost(true)
        }, { once: true })
      }}
      onPointerMissed={() => props.onSelect('api')}
    >
      <SceneContent {...props} />
    </Canvas>
  )
}
