export function TableHeader() {
  return (
    <div className="hidden md:grid grid-cols-[repeat(16,minmax(0,1fr))] gap-2 mb-3 text-[10px] font-black uppercase tracking-wider">
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Master ID
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        ID
      </div>
      <div className="col-span-2 bg-[#123050] text-white p-2 rounded-lg  flex items-center overflow-hidden min-w-0">
        Name
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Father Name
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Mobile
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Alt Phone
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Email
      </div>
      <div className="col-span-3 bg-[#123050] text-white p-2 rounded-lg font-black  flex items-center overflow-hidden min-w-0">
        Address
      </div>
      <div className="col-span-3 bg-[#123050] text-white p-2 rounded-lg font-black  flex items-center overflow-hidden min-w-0">
        Alt Address
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Year
      </div>
      <div className="col-span-1 bg-[#123050] text-white p-2 rounded-lg text-center  flex items-center justify-center overflow-hidden min-w-0">
        Action
      </div>
    </div>
  );
}
