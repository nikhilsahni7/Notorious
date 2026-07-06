export function TableHeader() {
  return (
    <div className="hidden md:grid grid-cols-[repeat(16,minmax(0,1fr))] gap-2 mb-3 text-[10px] font-black uppercase tracking-wider">
      <div className="col-span-1 bg-[#4A5568]/90 text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(74,85,104,0.1)] flex items-center justify-center">
        Master ID
      </div>
      <div className="col-span-1 bg-[#4A5568]/90 text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(74,85,104,0.1)] flex items-center justify-center">
        ID
      </div>
      <div className="col-span-2 bg-[#4A5568]/90 text-white p-2.5 rounded-lg shadow-[0_0_15px_rgba(74,85,104,0.1)] flex items-center">
        Name
      </div>
      <div className="col-span-1 bg-[#4A5568]/90 text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(74,85,104,0.1)] flex items-center justify-center">
        Father Name
      </div>
      <div className="col-span-1 bg-[#ED64A6] text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(237,100,166,0.2)] flex items-center justify-center">
        Mobile
      </div>
      <div className="col-span-1 bg-[#805AD5] text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(128,90,213,0.2)] flex items-center justify-center">
        Alt Phone
      </div>
      <div className="col-span-1 bg-[#ea580c] text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(234,88,12,0.2)] flex items-center justify-center">
        Email
      </div>
      <div className="col-span-3 bg-[#9AE6B4] text-gray-950 p-2.5 rounded-lg font-black shadow-[0_0_15px_rgba(154,230,180,0.2)] flex items-center">
        Address
      </div>
      <div className="col-span-3 bg-[#FC8181] text-white p-2.5 rounded-lg font-black shadow-[0_0_15px_rgba(252,129,129,0.2)] flex items-center">
        Alt Address
      </div>
      <div className="col-span-1 bg-[#4299E1] text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(66,153,225,0.2)] flex items-center justify-center">
        Year
      </div>
      <div className="col-span-1 bg-gray-600/90 text-white p-2.5 rounded-lg text-center shadow-[0_0_15px_rgba(75,85,99,0.1)] flex items-center justify-center">
        Action
      </div>
    </div>
  );
}
